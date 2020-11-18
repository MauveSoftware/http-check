package check

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	c := NewCheck(s.Client(), s.URL)
	c.AssertStatusCodeIn([]uint16{200})
	c.AssertBodyContains("valid")
	c.AssertHeaderExists("X-Test2", "bar")
	assert.Nil(t, c.Run())
}

func TestInvalidStausCode(t *testing.T) {
	s := mockServer(404, "the princess is in another castle", http.Header{})
	defer s.Close()

	c := NewCheck(s.Client(), s.URL)
	c.AssertStatusCodeIn([]uint16{200})
	err := c.Run()
	assert.EqualError(t, err, "Unexpected status code: 404 Not Found (expected: [200])")
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

func mockServer(status int, body string, headers http.Header) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for n, v := range headers {
			rw.Header().Add(n, v[0])
		}

		rw.WriteHeader(status)
		rw.Write([]byte(body))
	}))
}
