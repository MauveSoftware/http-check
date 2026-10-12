package check

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidReponse(t *testing.T) {
	s := mockServer(200, "this is a valid response", http.Header{
		"X-Test":  []string{"foo"},
		"X-Test2": []string{"bar"},
	})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL, WithDebug(os.Stdout))
	c.AssertStatusCodeIn([]uint32{200})
	c.AssertBodyContains("valid")
	c.AssertHeaderExists("X-Test2", "bar")
	assert.Nil(t, c.Run())
}

func TestInvalidStausCode(t *testing.T) {
	s := mockServer(404, "the princess is in another castle", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertStatusCodeIn([]uint32{200})
	err := c.Run()
	assert.EqualError(t, err, "unexpected status code: 404 Not Found (expected: [200])")
}

func TestTimeoutHandling(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		time.Sleep(10 * time.Millisecond)
		rw.WriteHeader(200)
		_, _ = rw.Write([]byte{})
	}))
	defer s.Close()

	cl := s.Client()
	cl.Timeout = time.Millisecond * 1

	c := NewCheck(cl, s.URL)
	err := c.Run()
	assert.EqualError(t, err, "timeout exceeded (1ms)")
}

func TestMissingHeader(t *testing.T) {
	s := mockServer(200, "", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertHeaderExists("X-Test", "Foo")
	err := c.Run()
	assert.EqualError(t, err, "expected header 'X-Test' with value 'Foo' (got: '')")
}

func TestHeaderMatches(t *testing.T) {
	s := mockServer(301, "", http.Header{
		"Location": []string{"https://example.com/en/start"},
	})
	defer s.Close()

	tests := []struct {
		name        string
		header      string
		regex       string
		expectedErr string
	}{
		{
			name:   "matching value",
			header: "Location",
			regex:  "^https://example\\.com/en/",
		},
		{
			name:        "non matching value",
			header:      "Location",
			regex:       "^https://example\\.com/de/",
			expectedErr: "regex '^https://example\\.com/de/' does not match header 'Location' (got: 'https://example.com/en/start')",
		},
		{
			name:        "missing header fails even if regex matches empty value",
			header:      "X-Missing",
			regex:       ".*",
			expectedErr: "header 'X-Missing' not found",
		},
		{
			name:        "invalid regex",
			header:      "Location",
			regex:       "[0-9]++",
			expectedErr: "invalid regex: error parsing regexp: invalid nested repetition operator: `++`",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cl := s.Client()
			cl.CheckRedirect = func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			}

			c := NewCheck(cl, s.URL)
			c.AssertHeaderMatches(test.header, test.regex)
			err := c.Run()

			if test.expectedErr == "" {
				assert.NoError(t, err, "header %s should match %q", test.header, test.regex)
				return
			}

			assert.EqualError(t, err, test.expectedErr, "unexpected result for header %s and regex %q", test.header, test.regex)
		})
	}
}

func TestInvalidBody(t *testing.T) {
	s := mockServer(200, "<body>Test</body>", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertBodyContains("Mauve")
	err := c.Run()
	assert.EqualError(t, err, "string 'Mauve' not found in body")
}

func TestInvalidBodyWithRegex(t *testing.T) {
	s := mockServer(200, "123456", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertBodyMatches("^\\d{5}$")
	err := c.Run()
	assert.EqualError(t, err, "regex '^\\d{5}$' does not match body")
}

func TestValidBodyWithRegex(t *testing.T) {
	s := mockServer(200, "12345", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertBodyMatches("^\\d{5}$")
	err := c.Run()
	assert.Nil(t, err)
}

func TestInvalidRegex(t *testing.T) {
	s := mockServer(200, "12345", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertBodyMatches("[0-9]++")
	err := c.Run()
	assert.NotNil(t, err)
}

func TestWithBasicAuth(t *testing.T) {
	c := NewCheck(http.DefaultClient, "www.mauve.de", WithBasicAuth("foo", "bar"))
	assert.Equal(t, c.username, "foo", "username")
	assert.Equal(t, c.password, "bar", "password")
}

func TestWithDebug(t *testing.T) {
	c := NewCheck(http.DefaultClient, "www.mauve.de", WithDebug(os.Stdout))
	assert.True(t, c.debug)
}

func TestInvalidPath(t *testing.T) {
	s := mockServer(200, "", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL+"xxx")
	c.AssertStatusCodeIn([]uint32{200})
	err := c.Run()
	assert.NotNil(t, err)
}

func TestInvalidUrl(t *testing.T) {
	s := mockServer(200, "", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL[1:])
	c.AssertStatusCodeIn([]uint32{200})
	err := c.Run()
	assert.NotNil(t, err)
}

func TestAssertCertificateExpireDaysWithoutCert(t *testing.T) {
	c := NewCheck(nil, "")
	c.AssertStatusCodeIn([]uint32{200})

	resp := &http.Response{
		StatusCode: http.StatusOK,
	}
	resp.TLS = &tls.ConnectionState{}

	c.AssertCertificateExpireDays(30 * 24 * time.Hour)
	err := c.validate(resp)

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "no certificate returned")
}

func TestAssertCertificateExpireDaysWithSoonExpiringCert(t *testing.T) {
	c := NewCheck(nil, "")
	c.AssertStatusCodeIn([]uint32{200})

	resp := &http.Response{
		StatusCode: http.StatusOK,
	}
	notAfter := time.Now().Add(10 * time.Minute)
	resp.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{
			{
				NotAfter: notAfter,
			},
		},
	}

	c.AssertCertificateExpireDays(30 * 24 * time.Hour)
	err := c.validate(resp)

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), fmt.Sprintf("certificate expires on %v", notAfter))
}

func mockServer(status int, body string, headers http.Header) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for n, v := range headers {
			rw.Header().Add(n, v[0])
		}

		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(body))
	}))
}
