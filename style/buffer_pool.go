package style

import (
	"strings"
	"sync"
)

var builderPool = sync.Pool{New: func() any { return new(strings.Builder) }}

func getBuilder() *strings.Builder {
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	return b
}

func putBuilder(b *strings.Builder) {
	if b == nil {
		return
	}
	b.Reset()
	builderPool.Put(b)
}
