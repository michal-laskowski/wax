
# wgo make dev
dev: 
	mkdir -p ./.coverage
	go run gotest.tools/gotestsum@latest -f testname -- ./... -race -count=1 -coverprofile=./.coverage/dev.out -covermode=atomic || true
	covreport -i ./.coverage/dev.out -o ./.coverage/dev.out.html
	
coverage:
	mkdir -p ./.coverage
	$(eval OUT := ./.coverage/$(shell date +%Y%m%d-%H%M%S).out)
	go run gotest.tools/gotestsum@latest -f testname -- ./... -race -count=1 -coverprofile=${OUT} -covermode=atomic
	covreport -i ${OUT} -o ${OUT}.html
	cp -u ${OUT}.html ./.coverage/last-result.html

tidy:
	go get -u
	go mod tidy -v

format:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	go run mvdan.cc/gofumpt@latest -w -l .

test:
	go run gotest.tools/gotestsum@latest -f testname -- ./... -race -shuffle=on

done: tidy format coverage
