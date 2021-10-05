package processes

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
	"github.com/ulfox/cinit/cinitd/models"
)

func (d *ProcessOperator) stopProcess(suid string) {
	if d.processPool[suid] == nil {
		return
	}

	d.processPool[suid].process.Kill()
	var terminated bool
	for {
		select {
		case <-time.After(10 * time.Second):
			d.processPool[suid].process.Signal(syscall.SIGKILL)
			return
		default:
			if d.processPool[suid].exitTime != nil {
				terminated = true
			} else {
				terminated = false
			}
			if terminated {
				return
			}
		}
	}
}

func (d *ProcessOperator) taskOperator(ctx context.Context, serviceQueue chan *Task, serviceChan *channels.Service, wg *sync.WaitGroup) {
	log := d.logger.WithFields(logrus.Fields{
		"Component": "ProcessPoolManager",
		"Part":      "TaskOperator",
	})
	var serviceInfoWG sync.WaitGroup
	for {
		select {
		case service := <-serviceChan.Action:

			serviceInfoWG.Add(1)
			go func(s chan models.ServiceAction, waitgroup *sync.WaitGroup, l *logrus.Entry) {
				defer waitgroup.Done()
				defer func(r chan models.ServiceAction) {
					close(r)
				}(s)

				var sa models.ServiceAction
			rLoop:
				for {
					select {
					case sa = <-s:
						break rLoop
					case <-time.After(10 * time.Second):
						l.Error("Done waiting for client")
						return
					}
				}

				if sa.SUID == "" {
					l.Error("Service SUID can not be empty")
				}

				if d.processPool[sa.SUID] == nil {
					sa.Status = "stopped"
					s <- sa
					return
				}

				if sa.T == "stop" || sa.T == "delete" {
					d.stopProcess(sa.SUID)
				}

				if sa.T == "delete" {
					d.Lock()
					delete(d.processPool, sa.SUID)
					d.Unlock()

					sa.Status = "deleted"
					s <- sa
					return
				}

				if d.processPool[sa.SUID].exitTime == nil {
					sa.Status = "running"
					sa.PID = fmt.Sprintf("%d", d.processPool[sa.SUID].process.Pid)
				} else {
					sa.Status = "stopped"
				}

				if sa.T == "start" {
					if sa.Status == "running" {
						l.Error("Can not start service. Already running...")
					} else {
						newService := models.Service{
							T:       sa.T,
							Name:    sa.Name,
							SUID:    sa.SUID,
							Command: d.processPool[sa.SUID].processInfo["args"][0],
							Args:    d.processPool[sa.SUID].processInfo["args"][1:],
						}
						d.Lock()
						delete(d.processPool, sa.SUID)
						d.Unlock()

						serviceChan.Push(newService)

						l.Info("Starting service...")

						var poolAdd bool
					poolAddLoop:
						for {
							select {
							case <-time.After(5 * time.Second):
								break poolAddLoop
							default:
								if d.processPool[sa.SUID] != nil {
									poolAdd = true
								} else {
									poolAdd = false
								}
								if poolAdd {
									break poolAddLoop
								}
							}
						}

						if !poolAdd {
							l.Error("Done waiting for service to be added in the pool")
							s <- sa
							return
						}

						var started bool
					poolStartLoop:
						for {
							select {
							case <-time.After(5 * time.Second):
								break poolStartLoop
							default:
								if d.processPool[sa.SUID].startTime != nil {
									started = true
								} else {
									started = false
								}
								if started {
									sa.Status = "running"
									sa.PID = fmt.Sprintf("%d", d.processPool[sa.SUID].process.Pid)
									break poolStartLoop
								}
							}
						}
					}
				}

				sa.ExitTime = d.processPool[sa.SUID].exitTime
				sa.StartTime = d.processPool[sa.SUID].startTime
				sa.ExitStatus = d.processPool[sa.SUID].exitStatus
				sa.Error = d.processPool[sa.SUID].err

				s <- sa
				l.Infof("Service Action %s finished", sa.Name)
			}(service, &serviceInfoWG, log)
		case <-ctx.Done():
			serviceInfoWG.Wait()
			wg.Done()
			log.Infof("Bye!")
			return
		default:
			time.Sleep(25 * time.Millisecond)
		}
	}
}
