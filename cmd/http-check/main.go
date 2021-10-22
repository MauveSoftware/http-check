package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/MauveSoftware/http-check/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "0.3.1"
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
	conn, err := grpc.Dial(
		*socketPath,
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		logrus.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewHttpCheckServiceClient(conn)

	req := &pb.Request{
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
	resp, err := c.Check(context.Background(), req)
	if err != nil {
		logrus.Fatal(err)
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
