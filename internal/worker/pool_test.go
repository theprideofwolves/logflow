package worker

import (
	"sync/atomic"
	"testing"
)

func TestStopDrainsQueue(t *testing.T) {
	var done int64
	d := NewDispatcher(8, 16, func([]byte) { atomic.AddInt64(&done, 1) })
	d.Start()
	for i := 0; i < 1000; i++ {
		d.Submit(Job{Payload: []byte("demo")})
	}
	d.Stop()
	if done != 1000 {
		t.Fatalf("lost jobs: processed=%d", done)
	}
}
