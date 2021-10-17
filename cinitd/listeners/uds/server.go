package uds

import (
	"context"
	"io"
	"net"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
)

// Server for managing a UDS listening service
type Server struct {
	sync.Mutex
	logger     *logrus.Logger
	ctx        context.Context
	wg         *sync.WaitGroup
	unixSocket string
	rcmd       *channels.Remote
}

// NewServerFactory for creating a new Server
func NewServerFactory(ctx context.Context, rcmd *channels.Remote, s string, l *logrus.Logger, wg *sync.WaitGroup) *Server {
	return &Server{
		logger:     l,
		unixSocket: s,
		rcmd:       rcmd,
		ctx:        ctx,
		wg:         wg,
	}
}

// ListenBackground for spawning a goroutine that reads UDS data and sending them over
// the remote command channel
func (s *Server) ListenBackground() {
	log := s.logger.WithFields(logrus.Fields{
		"Component": "SocketServer",
		"Part":      "Init",
	})
	log.Info("Server initializing")

	s.wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup, rcmd *channels.Remote, l *logrus.Logger) {
		defer wg.Done()

		log := s.logger.WithFields(logrus.Fields{
			"Component": "SocketServer",
		})

		var listenerWG sync.WaitGroup

		os.Remove(s.unixSocket)
		listener, err := net.Listen("unix", s.unixSocket)
		if err != nil {
			l.Fatal(err)
		}
		defer listener.Close()
		for {
			select {
			case <-ctx.Done():
				listenerWG.Wait()
				log.Info("Bye!")
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					l.Fatal(err)
				}

				listenerWG.Add(1)
				go func(c net.Conn, waitgroup *sync.WaitGroup, rcmd *channels.Remote, l *logrus.Logger) {
					defer waitgroup.Done()
					defer c.Close()

					dataRead := make([]byte, 0)
					for {
						buf := make([]byte, 512)
						b, err := c.Read(buf)
						if err != nil {
							if err != io.EOF {
								l.Fatal(err)
							}
							break
						}

						dataRead = append(dataRead, buf[:b]...)
					}

					if len(dataRead) == 0 {
						return
					}

					rChan := make(chan []byte)
					rcmd.Push(rChan)
					reply := <-rChan
					if string(reply) != "0x0" {
						log.Fatalf("rChan init was not 0x0")
					}

					rChan <- dataRead
					<-rChan
				}(conn, &listenerWG, rcmd, l)
			}
		}
	}(s.ctx, s.wg, s.rcmd, s.logger)
}
