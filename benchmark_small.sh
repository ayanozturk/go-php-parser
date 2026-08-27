#!/bin/bash
# Quick benchmark on small subset - finishes in ~30s

set -e

echo "Building..."
go build -o php-parser ./main.go

# Pick a smaller subset: first 1000 PHP files
echo "Finding files..."
SAMPLE_DIR="/tmp/php_sample"
rm -rf "$SAMPLE_DIR"
mkdir -p "$SAMPLE_DIR"

find /Volumes/RG-DOCK/rg_core -name "*.php" -type f 2>/dev/null | head -1000 | while read f; do
	mkdir -p "$SAMPLE_DIR/$(dirname "$f" | md5 -q | head -c 4)"
	cp "$f" "$SAMPLE_DIR/$(dirname "$f" | md5 -q | head -c 4)/"
done

FILES=$(find "$SAMPLE_DIR" -name "*.php" | wc -l)
LINES=$(find "$SAMPLE_DIR" -name "*.php" -exec wc -l {} \; 2>/dev/null | awk '{sum+=$1} END {print sum}')

echo "Sample: $FILES files, $LINES lines"

echo ""
echo "=== COLD RUN ==="
rm -rf ~/.cache/go-phpcs
START=$(date +%s%N)
./php-parser -config /dev/null "$SAMPLE_DIR" 2>&1 | grep -E "scanning|scanned|errors|HeapAlloc|Sys" | head -10
END=$(date +%s%N)
COLD_MS=$(( (END - START) / 1000000 ))
echo "Cold: ${COLD_MS}ms"

echo ""
echo "=== WARM RUN ==="
START=$(date +%s%N)
./php-parser -config /dev/null "$SAMPLE_DIR" 2>&1 | grep -E "scanning|scanned|errors|HeapAlloc|Sys" | head -10
END=$(date +%s%N)
WARM_MS=$(( (END - START) / 1000000 ))
echo "Warm: ${WARM_MS}ms"

echo ""
echo "=== CPU PROFILE ==="
rm -f cpu.prof
go tool pprof -top cpu.prof 2>&1 | head -20
