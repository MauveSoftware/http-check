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

type mock struct {
}

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
	assert.EqualError(t, err, "Unexpected status code: 404 Not Found (expected: [200])")
}

func TestTimeoutHandling(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		time.Sleep(10 * time.Millisecond)
		rw.WriteHeader(200)
		rw.Write([]byte{})
	}))
	defer s.Close()

	cl := s.Client()
	cl.Timeout = time.Millisecond * 1

	c := NewCheck(cl, s.URL)
	err := c.Run()
	assert.EqualError(t, err, "Timeout exceeded (1ms)")
}

func TestMissingHeader(t *testing.T) {
	s := mockServer(200, "", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertHeaderExists("X-Test", "Foo")
	err := c.Run()
	assert.EqualError(t, err, "Expected header 'X-Test' with value 'Foo'")
}

func TestInvalidBody(t *testing.T) {
	s := mockServer(200, "<body>Test</body>", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertBodyContains("Mauve")
	err := c.Run()
	assert.EqualError(t, err, "String 'Mauve' not found in body")
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
	assert.Equal(t, err.Error(), "No certificate returned")
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
	assert.Equal(t, err.Error(), fmt.Sprintf("Certificate expires on %v", notAfter))
}

func mockServer(status int, body string, headers http.Header) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for n, v := range headers {
			rw.Header().Add(n, v[0])
		}

		rw.WriteHeader(status)
		rw.Write([]byte(body))
	}))
}
