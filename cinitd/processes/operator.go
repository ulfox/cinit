package processes

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
	e "github.com/ulfox/cinit/cinitd/errors"
)

type erf = func(e interface{}, p ...interface{}) error

var (
	wrapErr erf = e.WrapErr
	wstatus syscall.WaitStatus
)

// ProcessOperator is the main process operator. It spawns zombieKiller, taskListener, handles
// tasks by forking them and terminates all processes on exit
type ProcessOperator struct {
	sync.Mutex
	logger             *logrus.Logger
	processPool        map[string]*ProcessHandler
	base               chan chan *Task
	task               chan *Task
	exitPO             <-chan bool
	ready              chan bool
	serviceChan        *channels.Service
	allowPoolExpanding bool
	watchAll           bool
	serviceLogDir      string
}

// NewProcessOperator creates, and returns a new ProcessOperator
func NewProcessOperator(exitPO <-chan bool, logger *logrus.Logger, watchAll bool, serviceChan *channels.Service, serviceLogDir string) *ProcessOperator {
	return &ProcessOperator{
		logger:             logger,
		task:               make(chan *Task),
		base:               make(chan chan *Task),
		processPool:        make(map[string]*ProcessHandler),
		ready:              make(chan bool),
		serviceChan:        serviceChan,
		exitPO:             exitPO,
		allowPoolExpanding: true,
		watchAll:           watchAll,
		serviceLogDir:      serviceLogDir,
	}
}

// Ready blocks until all ProcessOperator components have started
func (d *ProcessOperator) Ready() {
	d.Lock()
	<-d.ready
	d.Unlock()
}

// Close all channels created by ProcessOperator
func (d *ProcessOperator) CloseChannels() {
	close(d.task)
	close(d.base)
	close(d.ready)
}

func (d *ProcessOperator) expandForbid(wg *sync.WaitGroup) {
	defer wg.Done()

	log := d.logger.WithFields(logrus.Fields{
		"Component": "ProcessPoolManager",
		"Part":      "ProcessQueue",
	})

	d.allowPoolExpanding = false
	log.Warn("ProcessQueue expansion is now forbidden")
	var sigTermProcesses sync.WaitGroup
	sigTermProcesses.Add(1)
	d.terminateProcesses(&sigTermProcesses)
	sigTermProcesses.Wait()
	log.Warn("ProcessQueue base has been shrinked to 0")
}

func (d *ProcessOperator) expandPool(puid string, wg *sync.WaitGroup) {
	if d.allowPoolExpanding {
		d.logger.WithFields(logrus.Fields{
			"Component": "ProcessPoolManager",
			"Part":      "Expanding",
		}).Debug("Expanding ProcessPool")

		wg.Add(1)
		go func(puid string, waitgroup *sync.WaitGroup) {
			d.addProcessHandler(puid)
			waitgroup.Done()
		}(puid, wg)
		wg.Done()
	}
}

func (d *ProcessOperator) getProcesses() []*os.Process {
	processes := make([]*os.Process, 0)

	procFS, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil
	}

	cinitdPID := os.Getpid()
	for _, e := range procFS {
		p, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if p == 1 || p == cinitdPID {
			continue
		}

		_, err = os.Stat(fmt.Sprintf("/proc/%d/cmdline", p))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil
		}

		prc, err := os.FindProcess(p)
		if err != nil {
			continue
		}
		processes = append(processes, prc)
	}
	return processes
}

func (d *ProcessOperator) terminateProcesses(wg *sync.WaitGroup) {
	defer wg.Done()

	log := d.logger.WithFields(logrus.Fields{
		"Component": "ProcessPoolManager",
		"Part":      "ProcessQueue",
	})

	d.Lock()
	for _, j := range d.processPool {
		j.process.Signal(syscall.SIGTERM)
	}
	d.Unlock()

	if !d.watchAll {
		var notTerminated int
		d.Lock()
		if len(d.processPool) == 0 {
			return
		}
		d.Unlock()

		for {
			select {
			case <-time.After(60 * time.Second):
				for _, j := range d.processPool {
					j.process.Signal(syscall.SIGKILL)
				}
				return
			default:
				notTerminated = 0

				d.Lock()
				for _, k := range d.processPool {
					if k.exitTime == nil {
						notTerminated++
					}
				}
				d.Unlock()
				if notTerminated == 0 {
					return
				}
			}
		}
	}

	var sigTerm sync.WaitGroup
	log.Infof("Sending SIGTERM to all processes")
	for _, j := range d.getProcesses() {
		j.Signal(syscall.SIGTERM)
		sigTerm.Add(1)
		go func(p *os.Process, waitgroup *sync.WaitGroup) {
			p.Wait()
			waitgroup.Done()
		}(j, &sigTerm)
	}

	log.Infof("Checking if processes have terminated")
	for i := 0; i < 60; i++ {
		if len(d.getProcesses()) == 0 {
			return
		}
		time.Sleep(time.Second * 1)
	}

	log.Infof("Done waiting for processes. Sending SIGKILL to all processes")
	for _, j := range d.getProcesses() {
		j.Signal(syscall.SIGKILL)
		j.Wait()
	}
	sigTerm.Wait()
}

func (d *ProcessOperator) issueTask(task *Task, wg *sync.WaitGroup) {
	d.logger.WithFields(logrus.Fields{
		"Component": "ProcessPoolManager",
		"Part":      "Binding",
	}).Debug("Received task. Pushing to next available process handler")
	wg.Add(1)
	go func(waitgroup *sync.WaitGroup) {
		bindAvailableProcess := <-d.base
		bindAvailableProcess <- task
		waitgroup.Done()
	}(wg)
	wg.Done()
}

func (d *ProcessOperator) addProcessHandler(puid string) {
	log := d.logger.WithFields(logrus.Fields{
		"Component": "ProcessPoolManager",
		"Part":      "ProcessQueue",
	})

	process := NewProcessHandler(
		puid,
		d.base,
		d.logger,
	)

	go process.listenForTask()

	d.Lock()
	d.processPool[puid] = process
	d.Unlock()

	log.Debug(
		fmt.Sprintf(
			"ProcessHandler %s has been created",
			puid,
		),
	)

	<-process.finishedTask

	log.Debug(
		fmt.Sprintf(
			"ProcessHandler %s is shutting down",
			puid,
		),
	)

	process.done <- true

	if d.processPool[puid] != nil {
		d.Lock()
		if d.processPool[puid].process.Pid > 0 {
			d.processPool[puid].process.Signal(syscall.SIGKILL)
			d.processPool[puid].process.Wait()
			d.processPool[puid].process = &os.Process{Pid: -1}
		}
		d.Unlock()
	}

	<-process.done
	process.Close()
}

func (d *ProcessOperator) zKill(ctx context.Context, wg *sync.WaitGroup) {
	for {
		wpid, _ := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)

		if wpid > 0 {
			continue
		}

		select {
		case <-ctx.Done():
			_, _ = syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
			wg.Done()
			d.logger.WithFields(logrus.Fields{
				"Component": "ProcessPoolManager",
				"Part":      "ZombieKiller",
			}).Infof("Bye!")
			return
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// Init ProcessOperator to listen for new tasks
func (d *ProcessOperator) Init(wg *sync.WaitGroup) {
	var zKillWG, taskListenerWG, taskOperatorWG, expandForbidWG, issueTaskWG, processPoolWG sync.WaitGroup

	ctxZKill, cancelZKILL := context.WithCancel(context.Background())
	zKillWG.Add(1)
	go d.zKill(ctxZKill, &zKillWG)

	ctxTaskListener, cancelTaskListener := context.WithCancel(context.Background())
	taskListenerWG.Add(1)
	go d.taskListener(ctxTaskListener, d.task, d.serviceChan, &taskListenerWG)

	ctxTaskOperator, cancelTaskOperator := context.WithCancel(context.Background())
	taskOperatorWG.Add(1)
	go d.taskOperator(ctxTaskOperator, d.task, d.serviceChan, &taskOperatorWG)

	wg.Add(1)
	d.ready <- true
	for {
		select {
		case task := <-d.task:
			if d.allowPoolExpanding {
				processPoolWG.Add(1)
				d.expandPool(task.suid, &processPoolWG)

				issueTaskWG.Add(1)
				d.issueTask(task, &issueTaskWG)
			}
		case <-d.exitPO:
			cancelTaskListener()
			taskListenerWG.Wait()

			cancelTaskOperator()
			taskOperatorWG.Wait()

			expandForbidWG.Add(1)
			d.expandForbid(&expandForbidWG)

			issueTaskWG.Wait()
			processPoolWG.Wait()
			expandForbidWG.Wait()

			cancelZKILL()
			zKillWG.Wait()

			d.CloseChannels()
			wg.Done()
			d.logger.WithFields(logrus.Fields{
				"Component": "ProcessPoolManager",
			}).Infof("Bye!")
			return
		}
	}
}
