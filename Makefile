run:
	go run main.go

test:
	go test ./tests/...

coverage:
	go test -cover ./...
