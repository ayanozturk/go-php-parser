//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
)

func hostLoadAverages() (one, five, fifteen float64, ok bool) {
	content, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, false
	}
	fields := strings.Fields(string(content))
	if len(fields) < 3 {
		return 0, 0, 0, false
	}
	one, errOne := strconv.ParseFloat(fields[0], 64)
	five, errFive := strconv.ParseFloat(fields[1], 64)
	fifteen, errFifteen := strconv.ParseFloat(fields[2], 64)
	if errOne != nil || errFive != nil || errFifteen != nil {
		return 0, 0, 0, false
	}
	return one, five, fifteen, true
}
