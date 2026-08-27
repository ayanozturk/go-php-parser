#!/bin/bash

# Benchmark script: measure analysis performance
# Usage: ./benchmark.sh [cold|warm|both] [output-file]

set -e

MODE=${1:-both}
OUTPUT_FILE=${2:-benchmarks.json}

# Build binary
echo "Building..."
go build -o php-parser ./main.go

echo "Running benchmark ($MODE)..."

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
FILES=$(find /Volumes/RG-DOCK/rg_core -name "*.php" 2>/dev/null | wc -l)
LINES=$(find /Volumes/RG-DOCK/rg_core -name "*.php" 2>/dev/null -exec wc -l {} \; 2>/dev/null | awk '{sum+=$1} END {print sum}')

if [[ "$MODE" == "cold" || "$MODE" == "both" ]]; then
	echo "Cold run (no cache)..."
	rm -rf ~/.cache/go-phpcs
	START=$(date +%s.%N)
	./php-parser > /dev/null 2>&1 || true
	END=$(date +%s.%N)
	COLD_TIME=$(echo "$END - $START" | bc)
	echo "Cold time: ${COLD_TIME}s"
else
	COLD_TIME=0
fi

if [[ "$MODE" == "warm" || "$MODE" == "both" ]]; then
	echo "Warm run (with cache)..."
	START=$(date +%s.%N)
	./php-parser > /dev/null 2>&1 || true
	END=$(date +%s.%N)
	WARM_TIME=$(echo "$END - $START" | bc)
	echo "Warm time: ${WARM_TIME}s"
else
	WARM_TIME=0
fi

echo "Done. Files: $FILES, Lines: $LINES"
echo "Results: cold=${COLD_TIME}s, warm=${WARM_TIME}s"
