package main

import (
	"bufio"
	"flag"
	"io"
	"log"
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
		log.Println("exited with error:", err.Error())
	}()

	var (
		optSocket string
	)

	flag.StringVar(&optSocket, "socket", "logsock.sock", "logging socket")
	flag.Parse()

	if err = os.RemoveAll(optSocket); err != nil {
		return
	}

	var listener net.Listener
	if listener, err = net.Listen("unix", optSocket); err != nil {
		return
	}
	defer listener.Close()

	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		chErr <- serveListener(listener, os.Stdout)
	}()

	select {
	case err = <-chErr:
	case sig := <-chSig:
		log.Println("received signal:", sig)
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
