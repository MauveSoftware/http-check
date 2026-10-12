package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/MauveSoftware/http-check/internal/api"
	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"
)

const (
	version = "0.4.0"
)

var (
	verbose            = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()
	showVersion        = kingpin.Flag("version", "Show version info").Bool()
	protocol           = kingpin.Flag("protocol", "Protocol to use for the request").Default("https").String()
	host               = kingpin.Flag("host", "Hostname to use for the request").Short('h').String()
	path               = kingpin.Flag("path", "Path to use for the request").String()
	username           = kingpin.Flag("username", "Username to use for authentication").Short('u').String()
	password           = kingpin.Flag("password", "Password to use for authentication").Short('p').String()
	expectedStatusCode = kingpin.Flag("expect-status", "List of expected status codes").Short('s').Uint32List()
	expectedBody       = kingpin.Flag("expect-body-string", "Expected string in response body").Short('b').String()
	expectedBodyRegex  = kingpin.Flag("expect-body-regex", "Expected regex matching string in response body").Short('r').String()
	certExpireDays     = kingpin.Flag("cert-min-expire-days", "Minimum number of days until certificate expiration").Uint32()
	socketPath         = kingpin.Flag("socket-path", "Socket to use to communicate with the server performing the check").Default("/tmp/http-check.sock").String()
	insecure           = kingpin.Flag("insecure", "Allow invalid TLS certificaets (e.g. self signed)").Default("false").Bool()
	timeout            = kingpin.Flag("timeout", "Timeout for the check (should be less than the monitoring system's check timeout)").Default("55s").Duration()
	followRedirects    = kingpin.Flag("follow-redirects", "Follow HTTP redirects").Default("true").Bool()
	expectedHeaders    = kingpin.Flag("expect-header", "Expected response header in format 'Name: value' (repeatable)").Strings()
	expectedHdrRegexes = kingpin.Flag("expect-header-regex", "Expected response header matching regex in format 'Name: regex' (repeatable)").Strings()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	runCheck()
}

func runCheck() {
	headers, err := parseHeaders(*expectedHeaders)
	if err != nil {
		kingpin.Fatalf("%v", err)
	}

	headerRegexes, err := parseHeaderRegexes(*expectedHdrRegexes)
	if err != nil {
		kingpin.Fatalf("%v", err)
	}

	conn, err := grpc.NewClient(
		"unix:"+*socketPath,
		grpc.WithTransportCredentials(grpcinsecure.NewCredentials()),
	)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Error(err)
		}
	}()

	c := api.NewHttpCheckServiceClient(conn)

	req := &api.Request{
		Protocol:              *protocol,
		Host:                  *host,
		Path:                  *path,
		Username:              *username,
		Password:              *password,
		ExpectedStatusCode:    *expectedStatusCode,
		ExpectedBody:          *expectedBody,
		ExpectedBodyRegex:     *expectedBodyRegex,
		CertExpireDays:        *certExpireDays,
		Debug:                 *verbose,
		Insecure:              *insecure,
		DoNotFollowRedirects:  !*followRedirects,
		ExpectedHeaders:       headers,
		ExpectedHeaderRegexes: headerRegexes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := c.Check(ctx, req)
	if err != nil {
		fmt.Printf("CRITICAL - %s\n", err)
		os.Exit(2)
	}

	exitCode := 0
	status := "OK"

	if !resp.Success {
		status = "CRITICAL"
		exitCode = 2
	}

	fmt.Printf("%s - %s\n", status, resp.Message)

	if len(resp.DebugMessage) > 0 {
		fmt.Println(resp.DebugMessage)
	}

	os.Exit(exitCode)
}

// parseHeaders parses headers in curl style ("Name: value"). Only the first colon separates
// name and value, so values like URLs may contain colons. Header names must not contain
// whitespace (RFC 9110), which also catches a missing colon in "Location https://..."
func parseHeaders(raw []string) (map[string]string, error) {
	headers := make(map[string]string, len(raw))
	for _, h := range raw {
		name, value, found := strings.Cut(h, ":")
		name = strings.TrimSpace(name)
		if !found || name == "" || strings.ContainsAny(name, " \t") {
			return nil, fmt.Errorf("invalid header %q, expected format 'Name: value'", h)
		}

		headers[name] = strings.TrimSpace(value)
	}

	return headers, nil
}

// parseHeaderRegexes parses headers like parseHeaders and validates the regexes,
// so invalid patterns are reported before the check is sent to the server
func parseHeaderRegexes(raw []string) (map[string]string, error) {
	headers, err := parseHeaders(raw)
	if err != nil {
		return nil, err
	}

	for name, regex := range headers {
		if _, err := regexp.Compile(regex); err != nil {
			return nil, fmt.Errorf("invalid regex for header %q: %w", name, err)
		}
	}

	return headers, nil
}

func printVersion() {
	fmt.Println("http-check")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: 2020, Mauve Mailorder Software GmbH & Co. KG, Licensed under MIT license")
	fmt.Println("Easy to use replacement for nagios http_check")
}
