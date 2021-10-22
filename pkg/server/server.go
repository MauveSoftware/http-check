package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/MauveSoftware/http-check/pb"
)

// HTTPCheckServer runs HTTP checks. It provides an gRPC interface to receive check tasks
type HTTPCheckServer struct {
	workerCount uint32
	reqTimeout  time.Duration
	tlsTimeout  time.Duration
	ch          chan *task
}

// New creates a new server instance
func New(workerCount uint32, reqTimeout, tlsTimeout time.Duration) *HTTPCheckServer {
	s := &HTTPCheckServer{
		workerCount: workerCount,
		reqTimeout:  reqTimeout,
		tlsTimeout:  tlsTimeout,
		ch:          make(chan *task),
	}

	s.startWorkers()

	return s
}

func (s *HTTPCheckServer) startWorkers() {
	for i := 0; i < int(s.workerCount); i++ {
		w := &worker{
			id:         i + 1,
			cl:         s.newHttpClient(false),
			insecureCl: s.newHttpClient(true),
			ch:         s.ch,
		}
		go w.run()
	}
}

func (s *HTTPCheckServer) newHttpClient(insecure bool) *http.Client {
	var tr = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: s.reqTimeout,
		}).Dial,
		TLSHandshakeTimeout: s.tlsTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	return &http.Client{
		Transport: tr,
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
