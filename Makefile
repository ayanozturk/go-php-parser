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

# Fast, directional syntax optimization loop. This preloads a deterministic
# corpus sample and measures CST parsing, AST lowering, and the combined path
# separately. Override SYNTAX_BENCH_ROOT, SYNTAX_BENCH_FILES, or
# SYNTAX_BENCH_COUNT as needed. Full-corpus CV/accounting remains the release gate.
SYNTAX_BENCH_ROOT ?= $(CURDIR)/test_projects/wordpress-develop
SYNTAX_BENCH_FILES ?= 128
SYNTAX_BENCH_COUNT ?= 5
syntax-bench-quick:
	GOWORK=off GOMAXPROCS=1 SYNTAX_BENCH_DIR="$(SYNTAX_BENCH_ROOT)" SYNTAX_BENCH_FILES="$(SYNTAX_BENCH_FILES)" \
		go test ./syntax -run '^$$' -bench '^BenchmarkSyntaxCorpus' -benchtime=1x -count=$(SYNTAX_BENCH_COUNT) -benchmem -cpu=1

style:
	go run main.go style

build:
	go build -o tusk

compat-metrics: test-projects
	go run ./cmd/compat-metrics

# Corpus-level diagnostic compatibility with PHPStan. Example:
# make phpstan-compat PHPSTAN_COMPAT_ARGS="--root /project --paths src,tests --phpstan-bin /project/vendor/bin/phpstan"
phpstan-compat:
	go run ./cmd/phpstan-compat $(PHPSTAN_COMPAT_ARGS)

# Fetch large PHP projects for testing (includes vendor dependencies)
# Skips already-cloned projects so safe to re-run
test-projects:
	mkdir -p test_projects
	@cd test_projects && \
	if [ ! -d laravel ]; then echo "Fetching Laravel..." && git clone --depth 1 https://github.com/laravel/laravel.git laravel && cd laravel && composer install --no-dev --optimize-autoloader && cd ..; else echo "Laravel already present, skipping."; fi && \
	if [ ! -d symfony ]; then echo "Fetching Symfony..." && git clone --depth 1 https://github.com/symfony/symfony.git symfony && cd symfony && composer install --no-dev --optimize-autoloader && cd ..; else echo "Symfony already present, skipping."; fi && \
	if [ ! -d drupal ]; then echo "Fetching Drupal..." && git clone --depth 1 https://github.com/drupal/drupal.git drupal && cd drupal && composer install --no-dev --optimize-autoloader && cd ..; else echo "Drupal already present, skipping."; fi && \
	if [ ! -d composer-src ]; then echo "Fetching Composer..." && git clone --depth 1 https://github.com/composer/composer.git composer-src && cd composer-src && composer install --no-dev --optimize-autoloader && cd ..; else echo "Composer already present, skipping."; fi && \
	if [ ! -d phpunit ]; then echo "Fetching PHPUnit..." && git clone --depth 1 https://github.com/sebastianbergmann/phpunit.git phpunit && cd phpunit && composer install --no-dev --optimize-autoloader && cd ..; else echo "PHPUnit already present, skipping."; fi && \
	echo "Counting lines of code..." && \
	find . -name "*.php" -type f -exec wc -l {} + | tail -1

# Clean up test projects
clean-test-projects:
	rm -rf test_projects

# Run the parser on all test projects
test-on-projects:
	@if [ ! -d "test_projects" ]; then \
		echo "Test projects not found. Run 'make test-projects' first."; \
		exit 1; \
	fi
	find test_projects -name "*.php" -type f | head -100 | xargs -I {} sh -c 'echo "Processing {}" && go run main.go style "{}" || true'
