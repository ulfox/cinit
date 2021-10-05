package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
	"github.com/ulfox/cinit/cinitd/models"
)

// ServiceOperator for managing cinitd services
type ServiceOperator struct {
	sync.Mutex
	logger   *logrus.Logger
	exitSO   <-chan bool
	ready    chan bool
	services map[string]*models.Service
}

// NewProcessOperator creates, and returns a new ServiceOperator
func NewProcessOperator(exitSO <-chan bool, logger *logrus.Logger) *ServiceOperator {
	return &ServiceOperator{
		logger:   logger,
		exitSO:   exitSO,
		ready:    make(chan bool),
		services: make(map[string]*models.Service),
	}
}

// Ready blocks until all ServiceOperator components have started
func (d *ServiceOperator) Ready() {
	d.Lock()
	<-d.ready
	d.Unlock()
}

func (d *ServiceOperator) CloseChannels() {
	close(d.ready)
}

func (d *ServiceOperator) serviceListener(ctx context.Context, remote *channels.Remote, serviceChan *channels.Service, wg *sync.WaitGroup) {
	log := d.logger.WithFields(logrus.Fields{
		"Component": "ServiceOperator",
		"Part":      "ServiceListener",
	})

	var serviceWG sync.WaitGroup
	for {
		select {
		case rChan := <-remote.Data:
			serviceWG.Add(1)
			go func(r chan []byte, waitgroup *sync.WaitGroup, l *logrus.Entry) {

				defer waitgroup.Done()
				defer func(r chan []byte) {
					r <- []byte("0xF")
					time.Sleep(time.Millisecond * 100)
					close(r)
				}(rChan)

				r <- []byte("0x0")

				var args []byte
			rLoop:
				for {
					select {
					case args = <-r:
						break rLoop
					case <-time.After(10 * time.Second):
						l.Error("Done waiting for client")
						return
					}
				}

				s := &models.Service{}
				err := json.Unmarshal(args, s)
				if err != nil {
					l.Error(err)
					return
				}

				if s.T == "shutdown" {
					return
				}

				allowedTypes := []string{
					"restart", "start", "register", "stop", "status", "delete", "list",
				}
				var typeOK bool
				for _, j := range allowedTypes {
					if j == s.T {
						typeOK = true
					}
				}
				if !typeOK {
					l.Errorf("service action %s not supported", s.T)
					return
				}

				switch s.T {
				case "register":
					if d.services[s.Name] != nil {
						msg := fmt.Sprintf("service %s already exists", s.Name)
						l.Errorf(msg)
						r <- []byte(msg)
						break
					}
					s.SUID = uuid.New().String()
					serviceChan.Push(*s)

					r <- []byte("Service " + s.Name + " has been registered")
					d.services[s.Name] = s
				case "status", "delete", "stop", "start":
					data, err := d.serviceAction(s, serviceChan)
					if err != nil {
						r <- []byte(err.Error())
						l.Error(err)
						break
					}

					if s.T == "delete" {
						delete(d.services, s.Name)
					}

					r <- data
				case "list":
					var serviceList struct{ Services []string }
					for k := range d.services {
						serviceList.Services = append(serviceList.Services, k)
					}
					if len(serviceList.Services) == 0 {
						r <- []byte("No services")
						return
					}

					data, err := json.Marshal(serviceList)
					if err != nil {
						r <- []byte(err.Error())
						l.Error(err)
						break
					}
					r <- data
				}

			}(rChan, &serviceWG, log)
		case <-ctx.Done():
			serviceWG.Wait()
			wg.Done()
			log.Infof("Bye!")
			return
			// default:
			// 	time.Sleep(25 * time.Millisecond)
		}
	}
}

func (d *ServiceOperator) serviceAction(s *models.Service, serviceChan *channels.Service) ([]byte, error) {
	if d.services[s.Name] == nil {
		return nil, fmt.Errorf("Service " + s.Name + " does not exist")
	}
	siChan := make(chan models.ServiceAction)
	serviceChan.PushSA(siChan)

	si := models.ServiceAction{
		SUID: d.services[s.Name].SUID,
		Name: s.Name,
		T:    s.T,
	}

	siChan <- si
saLoop:
	for {
		select {
		case si = <-siChan:
			break saLoop
		case <-time.After(serviceChan.ActionTimeOut):
			msg := "done waiting for a response from ProcessPoolManager"
			return nil, fmt.Errorf(msg)
		}
	}

	data, err := json.Marshal(si)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Init ServiceOperator to listen to remote service commands
func (d *ServiceOperator) Init(remote *channels.Remote, serviceChan *channels.Service, wg *sync.WaitGroup) {
	log := d.logger.WithFields(logrus.Fields{
		"Component": "ServiceOperator",
	})

	var serviceListenerWG sync.WaitGroup

	ctxServiceListener, cancelServiceListener := context.WithCancel(context.Background())
	serviceListenerWG.Add(1)
	go d.serviceListener(ctxServiceListener, remote, serviceChan, &serviceListenerWG)

	wg.Add(1)
	d.ready <- true
	<-d.exitSO

	cancelServiceListener()
	serviceListenerWG.Wait()
	d.CloseChannels()
	wg.Done()
	log.Infof("Bye!")
}
