package processes

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
)

func (d *ProcessOperator) taskListener(ctx context.Context, serviceQueue chan *Task, serviceChan *channels.Service, wg *sync.WaitGroup) {
	log := d.logger.WithFields(logrus.Fields{
		"Component": "ProcessPoolManager",
		"Part":      "TaskListener",
	})
	for {
		select {
		case service := <-serviceChan.Data:
			if service.Command == "" {
				log.Errorf("Service %s command is empty", service.Name)
				break
			}

			if service.Name == "" || service.SUID == "" {
				log.Error("Service Name/SUID can not be empty")
				break
			}

			args := []string{service.Command}
			if len(service.Args) > 0 {
				args = append(args, service.Args...)
			}
			task := Task{
				suid: service.SUID,
				Name: service.Name,
				processInfo: map[string][]string{
					"args": args,
				},
				Exec: func() (*os.Process, error) {
					fork := newProcessFactory(
						service.SUID,
						d.serviceLogDir,
						"/",
						service.Name,
						uint32(os.Getuid()),
						uint32(os.Getgid()),
					)

					path, err := exec.LookPath(service.Command)
					if err != nil {
						return &os.Process{
							Pid: -1,
						}, err
					}

					prc, err := fork.exec(path, args)
					if err != nil {
						return &os.Process{
							Pid: -1,
						}, err
					}
					return prc, nil
				},
			}

			serviceQueue <- &task
			log.Infof("Registered new task")
		case <-ctx.Done():
			wg.Done()
			log.Infof("Bye!")
			return
		default:
			time.Sleep(25 * time.Millisecond)
		}
	}
}
