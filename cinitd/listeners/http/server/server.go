package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
	"github.com/ulfox/cinit/cinitd/listeners/http/server/router"
)

// Server for managing a HTTP listening service
type Server struct {
	sync.Mutex
	logger         *logrus.Logger
	ctx            context.Context
	wg             *sync.WaitGroup
	router         *router.Service
	port, listenAt string
}

func NewServerFactory(ctx context.Context, rcmd *channels.Remote, p, i string, l *logrus.Logger, wg *sync.WaitGroup) *Server {
	return &Server{
		logger:   l,
		port:     p,
		listenAt: i,
		ctx:      ctx,
		wg:       wg,
		router:   router.NewRouter(rcmd, l).UpdateRoutes(),
	}
}

// ListenBackground for spawning a goroutine that reads UDS data and sending them over
// the remote command channel
func (s *Server) ListenBackground() {
	log := s.logger.WithFields(logrus.Fields{
		"Component": "HTTP Server",
	})
	server := &http.Server{
		Addr:    s.listenAt + ":" + s.port,
		Handler: s.router.Router,
	}

	s.wg.Add(1)
	go func(wg *sync.WaitGroup, srv *http.Server) {
		err := srv.ListenAndServe()
		if err != nil {
			if err.Error() != "http: Server closed" {
				log.Error(err)
			}
		}
		wg.Done()
	}(s.wg, server)

	s.wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup, srv *http.Server, l *logrus.Entry) {
		for {
			select {
			case <-ctx.Done():
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := srv.Shutdown(ctx); err != nil {
					l.Fatal(err)
				}

				wg.Done()
				l.Info("Bye!")
				return
			default:
				time.Sleep(time.Millisecond * 50)
			}
		}
	}(s.ctx, s.wg, server, log)
}
