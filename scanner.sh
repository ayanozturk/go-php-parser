#!/bin/bash
count=0
find test_projects/symfony/vendor/twig/twig/src -name "*.php" | while read -r file; do
    output=$(go run . -debug style "$file" 2>&1)
    if [[ $? -ne 0 ]]; then
        echo "FILE: $file"
        echo "$output" | grep -v "exit status" | head -n 3
        echo "---"
        ((count++))
    fi
    if [[ $count -ge 5 ]]; then
        break
    fi
done
