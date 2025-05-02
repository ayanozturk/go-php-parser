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
	lsof -ti :8080 | xargs kill || true
	go tool pprof -http=:8080 go-phpcs cpu.prof

ast:
	go run main.go ast

test:
	go test ./...

coverage:
	go test -cover ./...

coverage-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

style:
	go run main.go style

build:
	go build -o go-phpcs
