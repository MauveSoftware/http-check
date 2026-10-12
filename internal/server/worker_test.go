package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MauveSoftware/http-check/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestRedirectHandling(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/old" {
			http.Redirect(rw, req, "/new", http.StatusMovedPermanently)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	w := &worker{cl: s.Client(), insecureCl: s.Client()}

	tests := []struct {
		name                 string
		doNotFollowRedirects bool
		expectedStatusCode   uint32
	}{
		{name: "follows redirects by default", doNotFollowRedirects: false, expectedStatusCode: http.StatusOK},
		{name: "does not follow redirects if requested", doNotFollowRedirects: true, expectedStatusCode: http.StatusMovedPermanently},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := w.processRequest(&api.Request{
				Protocol:             "http",
				Host:                 strings.TrimPrefix(s.URL, "http://"),
				Path:                 "/old",
				ExpectedStatusCode:   []uint32{test.expectedStatusCode},
				DoNotFollowRedirects: test.doNotFollowRedirects,
			})
			assert.True(t, resp.Success, "expected status code %d, got: %s", test.expectedStatusCode, resp.Message)
		})
	}
}

func TestExpectedHeaders(t *testing.T) {
	s := httptest.NewServer(http.RedirectHandler("/new", http.StatusMovedPermanently))
	defer s.Close()

	w := &worker{cl: s.Client(), insecureCl: s.Client()}

	tests := []struct {
		name            string
		expectedHeaders map[string]string
		wantSuccess     bool
	}{
		{name: "matching redirect target", expectedHeaders: map[string]string{"Location": "/new"}, wantSuccess: true},
		{name: "unexpected redirect target", expectedHeaders: map[string]string{"Location": "/other"}, wantSuccess: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := w.processRequest(&api.Request{
				Protocol:             "http",
				Host:                 strings.TrimPrefix(s.URL, "http://"),
				Path:                 "/old",
				DoNotFollowRedirects: true,
				ExpectedHeaders:      test.expectedHeaders,
			})
			assert.Equal(t, test.wantSuccess, resp.Success, "unexpected check result: %s", resp.Message)
		})
	}
}

func TestExpectedHeaderRegexes(t *testing.T) {
	s := httptest.NewServer(http.RedirectHandler("/new?session=4711", http.StatusFound))
	defer s.Close()

	w := &worker{cl: s.Client(), insecureCl: s.Client()}

	tests := []struct {
		name                  string
		expectedHeaderRegexes map[string]string
		wantSuccess           bool
	}{
		{name: "matching redirect target", expectedHeaderRegexes: map[string]string{"Location": "^/new\\?session=\\d+$"}, wantSuccess: true},
		{name: "unexpected redirect target", expectedHeaderRegexes: map[string]string{"Location": "^/other"}, wantSuccess: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := w.processRequest(&api.Request{
				Protocol:              "http",
				Host:                  strings.TrimPrefix(s.URL, "http://"),
				Path:                  "/old",
				DoNotFollowRedirects:  true,
				ExpectedHeaderRegexes: test.expectedHeaderRegexes,
			})
			assert.Equal(t, test.wantSuccess, resp.Success, "unexpected check result: %s", resp.Message)
		})
	}
}
