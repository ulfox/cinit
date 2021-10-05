package processes

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Task we define the body of the process we want to fork
type Task struct {
	suid, Name  string
	processInfo map[string][]string
	Exec        func() (*os.Process, error)
}

// ProcessHandler for executing processes
type ProcessHandler struct {
	puid         string
	process      *os.Process
	task         chan *Task
	base         chan chan *Task
	done         chan bool
	finishedTask chan bool
	processInfo  map[string][]string
	logger       *logrus.Logger
	startTime    *time.Time
	exitTime     *time.Time
	err          error
	exitStatus   string
}

// NewProcessHandler creates a new ProcessHandler. Essentially it creates a new Task and
// registers the task to the channel in order for listenForTask routines to pick it up and
// execute it
func NewProcessHandler(puid string, base chan chan *Task, logger *logrus.Logger) *ProcessHandler {
	process := &ProcessHandler{
		puid:         puid,
		task:         make(chan *Task),
		base:         base,
		done:         make(chan bool),
		finishedTask: make(chan bool),
		logger:       logger,
	}
	process.base <- process.task
	return process
}

func (w *ProcessHandler) Close() {
	close(w.task)
	close(w.done)
	close(w.finishedTask)
}

func (w *ProcessHandler) listenForTask() {
	for {
		select {
		case task := <-w.task:
			w.processInfo = task.processInfo
			prc, err := task.Exec()
			startTime := time.Now()
			w.startTime = &startTime
			w.process = prc
			if err != nil {
				w.logger.WithFields(logrus.Fields{
					"Component": "ProcessHandler",
					"Part":      "Fork",
				}).Error(err)
				w.err = err
				exitTime := time.Now()
				w.exitTime = &exitTime
				w.finishedTask <- true
				break
			}

			w.logger.WithFields(logrus.Fields{
				"Component": "ProcessHandler",
				"Part":      "Running",
				"PUID":      w.puid,
				"Name":      task.Name,
				"PID":       prc.Pid,
			},
			).Info("Task is being executed")

			// We are not releasing. Essentially we are not forking but spawning childrens
			// at the moment
			exit, err := prc.Wait()
			if err != nil {
				w.logger.WithFields(logrus.Fields{
					"Component": "ProcessHandler",
					"Part":      "Exit",
					"PUID":      w.puid,
					"Name":      task.Name,
				}).Error(err)
			} else {
				w.logger.WithFields(logrus.Fields{
					"Component": "ProcessHandler",
					"Part":      "Exit",
					"PUID":      w.puid,
					"Name":      task.Name,
				}).Infof("Task has finished: %s", exit)
			}
			exitTime := time.Now()
			w.exitTime = &exitTime
			w.err = err
			w.exitStatus = exit.String()
			w.finishedTask <- true
		case <-w.done:
			w.done <- true
			return
		}
	}
}
