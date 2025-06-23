package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/michal-laskowski/wax"
)

func main() {
	// specify where are you views
	viewsFS := os.DirFS("./views/")

	// instantiate engine
	viewResolver := wax.NewFsViewResolver(viewsFS)
	renderer := wax.New(viewResolver)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := renderer.Render(w,
			// Render view Hello
			"Hello",
			//Pass model (view params)
			map[string]any{"name": "John"})
		if err != nil {
			w.Write([]byte(err.Error()))
		}
	})

	fmt.Println("Listening on http://localhost:3000")
	http.ListenAndServe(":3000", nil)
}
