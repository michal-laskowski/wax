dev: 
	wgo go run gotest.tools/gotestsum@latest -f testname -- ./... -race -count=1 -coverprofile=${OUT} -covermode=atomic
	
done: tidy format coverage

coverage:
	mkdir -p ./.coverage
	$(eval OUT := ./.coverage/$(shell date +%Y%m%d-%H%M%S).out)
	go run gotest.tools/gotestsum@latest -f testname -- ./... -race -count=1 -coverprofile=${OUT} -covermode=atomic
	go tool cover -html=${OUT} -o ${OUT}.html

format:
	go run mvdan.cc/gofumpt@latest -w -l .

test:
	go run gotest.tools/gotestsum@latest -f testname -- ./... -race -count=1 -shuffle=on

tidy:
	go mod tidy -v
