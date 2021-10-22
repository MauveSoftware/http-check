package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MauveSoftware/http-check/pb"
	"github.com/MauveSoftware/http-check/pkg/check"
	"github.com/sirupsen/logrus"
)

type task struct {
	req *pb.Request
	ch  chan<- *pb.Response
}

type worker struct {
	id int
	cl *http.Client
	ch chan *task
}

func (w *worker) run() {
	for t := range w.ch {
		resp := w.processRequest(t.req)
		t.ch <- resp
	}
}

func (w *worker) processRequest(req *pb.Request) *pb.Response {
	logrus.Infof("#%d: Processing check for %s", w.id, req.Host)
	out := &strings.Builder{}
	c := w.checkForRequest(req, out)

	start := time.Now()
	err := c.Run()

	if err != nil {
		return &pb.Response{
			Success:      false,
			Message:      err.Error(),
			DebugMessage: out.String(),
		}
	}

	return &pb.Response{
		Success:      true,
		Message:      fmt.Sprintf("Request took %v", time.Since(start)),
		DebugMessage: out.String(),
	}
}

func (w *worker) checkForRequest(req *pb.Request, out io.Writer) *check.Check {
	opts := []check.Option{}

	if len(req.Username) > 0 {
		opts = append(opts, check.WithBasicAuth(req.Username, req.Password))
	}

	if req.Debug {
		opts = append(opts, check.WithDebug(out))
	}

	url := fmt.Sprintf("%s://%s%s", req.Protocol, req.Host, req.Path)
	c := check.NewCheck(w.cl, url, opts...)

	if len(req.ExpectedStatusCode) > 0 {
		c.AssertStatusCodeIn(req.ExpectedStatusCode)
	}

	if len(req.ExpectedBody) > 0 {
		c.AssertBodyContains(req.ExpectedBody)
	}

	if len(req.ExpectedBodyRegex) > 0 {
		c.AssertBodyMatches(req.ExpectedBodyRegex)
	}

	if req.CertExpireDays > 0 {
		c.AssertCertificateExpireDays(time.Duration(req.CertExpireDays) * 24 * time.Hour)
	}

	return c
}
