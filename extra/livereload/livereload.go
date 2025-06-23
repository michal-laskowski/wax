package livereload

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/radovskyb/watcher"
)

func StartLiveReload(config ...LiveReloadConfig) error {
	useConfig := defaultConfig
	if len(config) > 0 {
		useConfig = config[0]
	}

	if useConfig.ServerPort <= 0 {
		useConfig.ServerPort = defaultConfig.ServerPort
	}
	useConfig.toWatchAbs, _ = filepath.Abs(useConfig.WatchFolder)
	useConfig.lastMod = time.Now().UnixNano()
	useConfig.writeWait = 1 * time.Second
	useConfig.filePeriod = 1 * time.Second
	useConfig.upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	http.HandleFunc("/live-reload.js", useConfig.serveJS)
	http.HandleFunc("/ws", useConfig.serveWs)

	go useConfig.startWatcher()
	watchAddr := fmt.Sprintf(":%d", useConfig.ServerPort)
	return http.ListenAndServe(watchAddr, nil)

}

type LiveReloadConfig struct {
	ServerPort  int
	WatchFolder string

	toWatchAbs string
	lastMod    int64
	writeWait  time.Duration
	filePeriod time.Duration
	upgrader   *websocket.Upgrader
}

var defaultConfig = LiveReloadConfig{
	ServerPort:  8181,
	WatchFolder: ".",
}

func (this *LiveReloadConfig) writer(ws *websocket.Conn, lastMod int64) {

	fileTicker := time.NewTicker(this.filePeriod)
	defer func() {
		fileTicker.Stop()
		ws.Close()
	}()
	if lastMod < this.lastMod {
		lastMod = this.lastMod
		ws.SetWriteDeadline(time.Now().Add(this.writeWait))
		if err := ws.WriteMessage(websocket.TextMessage, []byte("reload")); err != nil {
			return
		}
	}
	for {
		select {
		case <-fileTicker.C:
			if lastMod < this.lastMod {
				//got change

				lastMod = this.lastMod
				ws.SetWriteDeadline(time.Now().Add(this.writeWait))
				if err := ws.WriteMessage(websocket.TextMessage, []byte("reload")); err != nil {
					return
				}
			} else {
				//no changes
				ws.SetReadDeadline(time.Now().Add(6 * time.Second))
				_, _, err := ws.ReadMessage()
				if err != nil {
					return
				}
			}
		}
	}
}

func (this *LiveReloadConfig) startWatcher() {
	w := watcher.New()

	w.SetMaxEvents(1)
	w.FilterOps(watcher.Write)
	this.lastMod = time.Now().UnixNano()

	go func() {
		for {
			select {
			case <-w.Event:
				this.lastMod = time.Now().UnixNano()
			case err := <-w.Error:
				panic(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(this.WatchFolder); err != nil {
		panic(err)
	}

	if err := w.Start(time.Millisecond * 100); err != nil {
		panic(err)
	}

	defer func() {
		w.Close()
	}()
}

func (this *LiveReloadConfig) serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := this.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			panic(err)
		}
		return
	}
	queryLastMod := r.URL.Query().Get("lastMod")
	lastMod, err := strconv.ParseInt(queryLastMod, 10, 64)
	if err != nil {
		lastMod = 0
	}
	go this.writer(ws, lastMod)
}

func (this *LiveReloadConfig) serveJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	time := strconv.FormatInt(time.Now().UnixNano(), 10)
	w.Write([]byte(fmt.Sprintf(clientCode, r.Host, time)))
}

const clientCode = `
if ('WebSocket' in window) {
    (function() {

        function connect(){
            var protocol = window.location.protocol === 'http:' ? 'ws://' : 'wss://';
            //var address = protocol + window.location.host + window.location.pathname + 'ws?lastMod=___s';
            var address = "ws://%s/ws?lastMod=%s";
            var socket = new WebSocket(address);

            socket.onmessage = function(msg) {
                if (msg.data == 'reload') window.location.reload();
            };

            socket.onclose = function(evt) {
                console.log(evt,'Connection closed');
                if (document.hidden){
                    console.log("try reconnect livereload") 
                    setTimeout(()=> connect(), 5000)
                } else { 
                    setTimeout(()=> connect(), 5000)
                }
            }

            socket.onerror = function(evt) {
                console.log(evt, 'Connection error');
            }
                        
            const ping = (()=>{
                if (socket.readyState === 1)
                    socket.send("ping")
                    setTimeout(ping, 2000)
                })
            ping()
            console.log('Live reload enabled.');       
        }

        connect()
    })();
}
`
