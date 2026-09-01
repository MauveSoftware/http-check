package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MauveSoftware/http-check/internal/api"
	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"
)

const (
	version = "0.3.5"
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
		Protocol:           *protocol,
		Host:               *host,
		Path:               *path,
		Username:           *username,
		Password:           *password,
		ExpectedStatusCode: *expectedStatusCode,
		ExpectedBody:       *expectedBody,
		ExpectedBodyRegex:  *expectedBodyRegex,
		CertExpireDays:     *certExpireDays,
		Debug:              *verbose,
		Insecure:           *insecure,
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

func printVersion() {
	fmt.Println("http-check")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: 2020, Mauve Mailorder Software GmbH & Co. KG, Licensed under MIT license")
	fmt.Println("Easy to use replacement for nagios http_check")
}
