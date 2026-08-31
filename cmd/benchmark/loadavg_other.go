//go:build !linux

package main

func hostLoadAverages() (one, five, fifteen float64, ok bool) {
	return 0, 0, 0, false
}
