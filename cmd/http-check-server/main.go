package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"

	"github.com/MauveSoftware/http-check/pb"
	"github.com/MauveSoftware/http-check/server"
	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "0.3.0"
)

var (
	showVersion = kingpin.Flag("version", "Show version info").Bool()
	workerCount = kingpin.Flag("worker-count", "Number of workers processing http checks in parallel").Default("25").Uint32()
	timeout     = kingpin.Flag("timeout", "Timeout after a connection attempt will be cancelled").Default("10s").Duration()
	socketPath  = kingpin.Flag("socket-path", "Socket to create to listen for check requests").Default("/tmp/http-check.sock").String()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	lis, err := openSocket()
	if err != nil {
		logrus.Fatal(err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	logrus.Infof("Starting %d workers", *workerCount)
	s := server.New(*workerCount, server.WithTimeout(*timeout))
	pb.RegisterHttpCheckServiceServer(srv, s)

	logrus.Infof("Listen for connections on socket %s", *socketPath)
	go logrus.Error(srv.Serve(lis))

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
	logrus.Info("Shutting down server")
	cleanupSocket()
}

func openSocket() (net.Listener, error) {
	cleanupSocket()
	lis, err := net.Listen("unix", *socketPath)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to listen on socket")
	}

	return lis, nil
}

func cleanupSocket() {
	_, err := os.Stat(*socketPath)
	if os.IsNotExist(err) {
		return
	}

	err = os.Remove(*socketPath)
	if err != nil {
		logrus.Error(err)
	}
}

func printVersion() {
	fmt.Println("http-check-server")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: 2021, Mauve Mailorder Software GmbH & Co. KG, Licensed under MIT license")
	fmt.Println("Server component for Mauve http-check")
}
