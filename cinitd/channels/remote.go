package channels

import (
	"sync"
	"time"
)

type Remote struct {
	sync.Mutex
	Data        chan chan []byte
	t           bool
	DataTimeOut time.Duration
}

func NewRemoteChannel(dt int) *Remote {
	return &Remote{
		Data:        make(chan chan []byte),
		DataTimeOut: time.Duration(dt) * time.Second,
	}
}

func (r *Remote) Push(d chan []byte) {
	r.Lock()
	r.Data <- d
	r.Unlock()
}

func (r *Remote) Term(t bool) {
	r.t = t
	if r.t {
		r.Close()
	}
}

func (r *Remote) Close() {
	r.Lock()
	close(r.Data)
	r.Unlock()
}
