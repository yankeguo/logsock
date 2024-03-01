package main

import (
	"bufio"
	"flag"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var err error

	defer func() {
		if err == nil {
			return
		}
		os.Exit(1)
	}()

	var (
		optLog    string
		optSocket string
	)

	flag.StringVar(&optLog, "log", "log.txt", "log file")
	flag.StringVar(&optSocket, "socket", "logsock.sock", "logging socket")
	flag.Parse()

	if err = os.RemoveAll(optSocket); err != nil {
		return
	}

	var out *os.File
	if out, err = os.OpenFile(optLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
		return
	}
	defer out.Close()

	var listener net.Listener
	if listener, err = net.Listen("unix", optSocket); err != nil {
		return
	}
	defer listener.Close()

	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		chErr <- serveListener(listener, out)
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
