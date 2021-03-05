package server

import (
	"context"
	"net/http"
	"time"

	"github.com/MauveSoftware/http-check/pb"
)

// HTTPCheckServer runs HTTP checks. It provides an gRPC interface to receive check tasks
type HTTPCheckServer struct {
	cl          *http.Client
	workerCount uint32
	ch          chan *task
}

// Option specifies options for the server
type Option func(*HTTPCheckServer)

// WithTimeout specifies the timeout for each HTTP request
func WithTimeout(t time.Duration) Option {
	return func(s *HTTPCheckServer) {
		s.cl.Timeout = t
	}
}

// New creates a new server instance
func New(workerCount uint32, opts ...Option) *HTTPCheckServer {
	s := &HTTPCheckServer{
		cl:          &http.Client{},
		workerCount: workerCount,
		ch:          make(chan *task),
	}

	for _, opt := range opts {
		opt(s)
	}
	s.startWorkers()

	return s
}

func (s *HTTPCheckServer) startWorkers() {
	for i := 0; i < int(s.workerCount); i++ {
		w := &worker{
			id: i + 1,
			cl: s.cl,
			ch: s.ch,
		}
		go w.run()
	}
}

// Check performs a http check and returns the check result
func (s *HTTPCheckServer) Check(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	respCh := make(chan *pb.Response, 1)
	s.ch <- &task{
		req: in,
		ch:  respCh,
	}

	resp := <-respCh
	return resp, nil
}
