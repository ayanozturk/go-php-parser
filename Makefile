# Usage:
#   make run ARGS="-o results.log"
#   make run ARGS="--output=results.log"
run:
	go run main.go $(ARGS)

ast:
	go run main.go ast

test:
	go test ./...

coverage:
	go test -cover ./...

style:
	go run main.go style

build:
	go build -o go-phpcs
