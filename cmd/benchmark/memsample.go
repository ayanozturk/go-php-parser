package main

import (
	"runtime/metrics"
	"sync/atomic"
	"time"
)

const processMemoryMetric = "/memory/classes/total:bytes"

// startMemSampler tracks peak process memory without stopping the world.
// The previous implementation called runtime.ReadMemStats on a 20ms ticker,
// which is a STW operation and injected measurable variance into process-cold
// timings. runtime/metrics.Read is the cheap, non-STW equivalent.
func startMemSampler() func() int64 {
	var peak atomic.Int64
	stop := make(chan struct{})
	done := make(chan struct{})
	sample := []metrics.Sample{{Name: processMemoryMetric}}
	read := func() {
		metrics.Read(sample)
		if sample[0].Value.Kind() != metrics.KindUint64 {
			return
		}
		value := int64(sample[0].Value.Uint64())
		for {
			old := peak.Load()
			if value <= old || peak.CompareAndSwap(old, value) {
				return
			}
		}
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		read()
		for {
			select {
			case <-stop:
				read()
				return
			case <-ticker.C:
				read()
			}
		}
	}()
	return func() int64 {
		close(stop)
		<-done
		return peak.Load()
	}
}
