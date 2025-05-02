# Usage:
#   make run ARGS="-o results.log"
#   make run ARGS="--output=results.log"
run:
	go run main.go $(ARGS)

# Run style check with profiling enabled. Outputs cpu.prof and mem.prof
profile:
	go run main.go --profile style

# Visualize the cpu.prof profile in the browser (pprof web UI)
profile-web:
	go tool pprof -http=:8080 go-phpcs cpu.prof

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
