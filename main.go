package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	outStdout = "-"
)

func main() {
	var err error

	defer func() {
		if err == nil {
			return
		}
		log.Println("exited with error:", err.Error())
		os.Exit(1)
	}()

	var (
		optOut    string
		optListen string
	)

	flag.StringVar(&optOut, "out", outStdout, "output file, use - for stdout")
	flag.StringVar(&optListen, "listen", "/var/log/log.sock", "address to listen on (support both tcp and unix socket)")
	flag.Parse()

	var out *os.File
	if optOut == outStdout {
		out = os.Stdout
	} else {
		if out, err = os.OpenFile(optOut, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
			return
		}
		defer out.Close()
	}

	var lis net.Listener
	if strings.Contains(optListen, ":") {
		if lis, err = net.Listen("tcp", optListen); err != nil {
			return
		}
	} else {
		if err = os.RemoveAll(optListen); err != nil {
			return
		}
		if lis, err = net.Listen("unix", optListen); err != nil {
			return
		}
	}
	defer lis.Close()

	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		chErr <- serveListener(lis, out)
	}()

	select {
	case err = <-chErr:
	case <-chSig:
	}
}

// WriterWithSync is an interface that extends io.Writer with a Sync method.
type WriterWithSync interface {
	io.Writer

	Sync() error
}

func serveListener(listener net.Listener, w WriterWithSync) (err error) {
	lines := make(chan []byte)
	defer close(lines)

	go func() {
		for line := range lines {
			w.Write(line)
			w.Sync()
		}
	}()

	for {
		var conn net.Conn
		if conn, err = listener.Accept(); err != nil {
			return
		}
		go handleConn(conn, lines)
	}
}

func handleConn(conn net.Conn, lines chan []byte) {
	defer conn.Close()

	br := bufio.NewReader(conn)

	for {
		line, err := br.ReadBytes('\n')

		if len(line) > 0 {
			if line[len(line)-1] != '\n' {
				line = append(line, '\n')
			}
			lines <- line
		}

		if err != nil {
			return
		}
	}
}
