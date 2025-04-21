run:
	go run main.go

ast:
	go run main.go ast

test:
	go test ./...

coverage:
	go test -cover ./...

style:
	go run main.go style
