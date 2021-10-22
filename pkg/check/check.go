package check

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Option configures a check
type Option func(*Check)

// WithBasicAuth defines basic auth parameters used by the check
func WithBasicAuth(username, password string) Option {
	return func(c *Check) {
		c.username = username
		c.password = password
	}
}

// WithDebug enables debug output
func WithDebug(w io.Writer) Option {
	return func(c *Check) {
		c.debug = true
		c.debugWriter = w
	}
}

// Check executes a web request and validates the response against a set of defined assertions
type Check struct {
	client      *http.Client
	url         string
	username    string
	password    string
	assertions  []assertion
	debug       bool
	debugWriter io.Writer
}

type assertion func(*http.Response) error

// NewCheck creates a new Check instance
func NewCheck(client *http.Client, url string, opts ...Option) *Check {
	c := &Check{
		client: client,
		url:    url,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Run executes a check
func (c *Check) Run() error {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return errors.Wrap(err, "Could not create request")
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("User-Agent", "mauve/http-check")

	resp, err := c.client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "Timeout") {
			return fmt.Errorf("Timeout exceeded (%v)", c.client.Timeout)
		}

		return err
	}
	defer resp.Body.Close()

	if c.debug {
		fmt.Fprintln(c.debugWriter, "Status: "+resp.Status)
		resp.Header.Write(c.debugWriter)
		fmt.Fprintln(c.debugWriter, "")
	}

	return c.validate(resp)
}

// AssertStatusCodeIn tests if status code is in expected range
func (c *Check) AssertStatusCodeIn(codes []uint32) {
	c.assertions = append(c.assertions, func(resp *http.Response) error {
		for _, c := range codes {
			if uint32(resp.StatusCode) == c {
				return nil
			}
		}

		return fmt.Errorf("Unexpected status code: %s (expected: %v)", resp.Status, codes)
	})
}

// AssertHeaderExists tests if a specified header with specific value exists
func (c *Check) AssertHeaderExists(name, value string) {
	c.assertions = append(c.assertions, func(resp *http.Response) error {
		h := resp.Header.Get(name)
		if h != value {
			return fmt.Errorf("Expected header '%s' with value '%v'", name, value)
		}

		return nil
	})
}

// AssertBodyContains tests if the body contains the specified string
func (c *Check) AssertBodyContains(s string) {
	c.assertions = append(c.assertions, func(resp *http.Response) error {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Could not read body")
		}

		if !strings.Contains(string(b), s) {
			return fmt.Errorf("String '%s' not found in body", s)
		}

		return nil
	})
}

// AssertBodyMatches tests if the body matches the specified regex
func (c *Check) AssertBodyMatches(regex string) {
	c.assertions = append(c.assertions, func(resp *http.Response) error {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Could not read body")
		}

		r, err := regexp.Compile(regex)
		if err != nil {
			return errors.Wrap(err, "Invalid regex")
		}

		if !r.Match(b) {
			return fmt.Errorf("Regex '%s' does not match body", regex)
		}

		return nil
	})
}

// AssertCertificateExpireDays tests the days until expiration of the returned certificate
func (c *Check) AssertCertificateExpireDays(d time.Duration) {
	c.assertions = append(c.assertions, func(resp *http.Response) error {
		if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
			return fmt.Errorf("No certificate returned")
		}

		first := resp.TLS.PeerCertificates[0]
		min := time.Now().Add(d)
		if !first.NotAfter.After(min) {
			return fmt.Errorf("Certificate expires on %v", first.NotAfter)
		}

		return nil
	})
}

func (c *Check) validate(resp *http.Response) error {
	for _, a := range c.assertions {
		if err := a(resp); err != nil {
			return err
		}
	}

	return nil
}
