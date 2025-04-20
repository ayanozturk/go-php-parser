run:
	go run main.go

test:
	go test ./...

coverage:
	go test -cover ./...

style:
	go run main.go style
