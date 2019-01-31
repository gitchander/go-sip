package transport

import (
	"sync/atomic"
)

type SyncBool struct {
	x *int32
}

func NewSyncBool(v bool) *SyncBool {
	sb := &SyncBool{
		x: new(int32),
	}
	sb.Set(v)
	return sb
}

func (sb *SyncBool) Set(v bool) {
	var x int32
	if v {
		x = 1
	}
	atomic.StoreInt32(sb.x, x)
}

func (sb *SyncBool) Get() bool {
	x := atomic.LoadInt32(sb.x)
	return x != 0
}
