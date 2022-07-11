package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MauveSoftware/http-check/internal/api"
	"github.com/MauveSoftware/http-check/pkg/check"
	"github.com/sirupsen/logrus"
)

type task struct {
	req *api.Request
	ch  chan<- *api.Response
}

type worker struct {
	id         int
	cl         *http.Client
	insecureCl *http.Client
	ch         chan *task
}

func (w *worker) run() {
	for t := range w.ch {
		resp := w.processRequest(t.req)
		t.ch <- resp
	}
}

func (w *worker) processRequest(req *api.Request) *api.Response {
	logrus.Infof("#%d: Processing check for %s", w.id, req.Host)
	out := &strings.Builder{}
	c := w.checkForRequest(req, out)

	start := time.Now()
	err := c.Run()

	if err != nil {
		return &api.Response{
			Success:      false,
			Message:      err.Error(),
			DebugMessage: out.String(),
		}
	}

	return &api.Response{
		Success:      true,
		Message:      fmt.Sprintf("Request took %v", time.Since(start)),
		DebugMessage: out.String(),
	}
}

func (w *worker) checkForRequest(req *api.Request, out io.Writer) *check.Check {
	opts := []check.Option{}

	if len(req.Username) > 0 {
		opts = append(opts, check.WithBasicAuth(req.Username, req.Password))
	}

	if req.Debug {
		opts = append(opts, check.WithDebug(out))
	}

	url := fmt.Sprintf("%s://%s%s", req.Protocol, req.Host, req.Path)

	cl := w.cl
	if req.Insecure {
		cl = w.insecureCl
	}

	c := check.NewCheck(cl, url, opts...)

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
