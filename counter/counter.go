package counter

import (
	"sync/atomic"
)

type Counter int32

func (c *Counter) Add(val int32) {
	atomic.AddInt32((*int32)(c), val)
}

func (c *Counter) Set(val int32) {
	atomic.StoreInt32((*int32)(c), val)
}

func (c *Counter) Get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

func (c *Counter) Inc() {
	c.Add(1)
}

func (c *Counter) Dec() {
	c.Add(-1)
}

func (c *Counter) Stop() {
	c.Set(0)
}
