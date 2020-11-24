package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/MauveSoftware/http-check/check"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "0.1.1"
)

var (
	verbose            = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()
	showVersion        = kingpin.Flag("version", "Show version info").Bool()
	protocol           = kingpin.Flag("protocol", "Protocol to use for the request").Default("https").String()
	host               = kingpin.Flag("host", "Hostname to use for the request").Short('h').String()
	path               = kingpin.Flag("path", "Path to use for the request").String()
	username           = kingpin.Flag("username", "Username to use for authentication").Short('u').String()
	password           = kingpin.Flag("password", "Password to use for authentication").Short('p').String()
	expectedStatusCode = kingpin.Flag("expect-status", "List of expected status codes").Short('s').Uint16List()
	expectedBody       = kingpin.Flag("expect-body-string", "Expected string in response body").Short('b').String()
	timeout            = kingpin.Flag("timeout", "Timeout after a connection attempt will be cancelled").Default("10s").Duration()
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
	opts := []check.Option{}

	if len(*username) > 0 {
		opts = append(opts, check.WithBasicAuth(*username, *password))
	}

	if *verbose {
		opts = append(opts, check.WithDebug())
	}

	url := fmt.Sprintf("%s://%s%s", *protocol, *host, *path)
	cl := &http.Client{}
	cl.Timeout = *timeout
	c := check.NewCheck(cl, url, opts...)

	if len(*expectedStatusCode) > 0 {
		c.AssertStatusCodeIn(*expectedStatusCode)
	}

	if len(*expectedBody) > 0 {
		c.AssertBodyContains(*expectedBody)
	}

	start := time.Now()
	err := c.Run()
	if err != nil {
		fmt.Println("CRITICAL - " + err.Error())
		os.Exit(2)
	}

	fmt.Println("OK - Request took", time.Since(start))
}

func printVersion() {
	fmt.Println("http-check")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: 2020, Mauve Mailorder Software GmbH & Co. KG, Licensed under MIT license")
	fmt.Println("Easy to use replacement for nagios http_check")
}
