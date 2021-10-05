package processes

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

type process struct {
	stdout, stderr                   *os.File
	uid, gid                         uint32
	processDir, taskID, logDir, name string
	process                          *os.Process
}

func newProcessFactory(taskID, logDir, dir, name string, uid, gid uint32) *process {
	return &process{
		taskID:     taskID,
		logDir:     logDir,
		uid:        uid,
		gid:        gid,
		processDir: dir,
		name:       name,
	}
}

func (p *process) exec(path string, args []string) (*os.Process, error) {
	err := p.makeDirs(p.logDir, 0760)
	if err != nil {
		return &os.Process{
			Pid: -1,
		}, wrapErr(err)
	}

	p.stdout, err = os.OpenFile(
		fmt.Sprintf("%s/%s-out.log", p.logDir, p.name),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666,
	)
	if err != nil {
		return &os.Process{
			Pid: -1,
		}, wrapErr(err)
	}

	p.stderr, err = os.OpenFile(
		fmt.Sprintf("%s/%s-err.log", p.logDir, p.name),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666,
	)
	if err != nil {
		return &os.Process{
			Pid: -1,
		}, wrapErr(err)
	}

	defer p.stdout.Close()
	defer p.stderr.Close()

	if err != nil {
		return &os.Process{
			Pid: -1,
		}, wrapErr(err)
	}
	prc, err := os.StartProcess(
		path,
		args,
		&os.ProcAttr{
			Dir: p.processDir,
			Env: os.Environ(),
			Files: []*os.File{
				nil,
				p.stdout,
				p.stderr,
			},
			Sys: &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid:         p.uid,
					Gid:         p.gid,
					Groups:      nil,
					NoSetGroups: true,
				},
				Setsid: true,
			},
		},
	)
	if err != nil {
		return prc, wrapErr(err)
	}
	p.process = prc
	return prc, nil
}

func (p *process) makeDirs(path string, m os.FileMode) error {
	if path == "/" {
		return nil
	}

	f, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return wrapErr(err)
		}
	} else {
		if f.IsDir() {
			return nil
		}
	}

	path = strings.TrimSuffix(path, "/")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, m)
		if err != nil {
			return wrapErr(err)
		}
	}
	return nil
}
