package channels

import (
	"sync"
	"time"

	"github.com/ulfox/cinit/cinitd/models"
)

type Service struct {
	sync.Mutex
	Data                       chan models.Service
	Action                     chan chan models.ServiceAction
	DataTimeOut, ActionTimeOut time.Duration
}

func NewServiceChannel(dt, at int) *Service {
	return &Service{
		Data:          make(chan models.Service),
		DataTimeOut:   time.Duration(dt) * time.Second,
		Action:        make(chan chan models.ServiceAction),
		ActionTimeOut: time.Duration(at) * time.Second,
	}
}

func (r *Service) Push(d models.Service) {
	r.Lock()
	r.Data <- d
	r.Unlock()
}

func (r *Service) PushSA(d chan models.ServiceAction) {
	r.Lock()
	r.Action <- d
	r.Unlock()
}

func (r *Service) Close() {
	r.Lock()
	close(r.Data)
	close(r.Action)
	r.Unlock()
}
