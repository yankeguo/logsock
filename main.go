package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	ArgOutStdout = "-"

	Newline = '\n'
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
		argOut    string
		argListen string
	)

	flag.StringVar(&argOut, "out", ArgOutStdout, "output file, use - for stdout")
	flag.StringVar(&argListen, "listen", "/var/log/log.sock", "address to listen on (support both tcp and unix socket)")
	flag.Parse()

	var out *os.File

	if argOut == ArgOutStdout {
		out = os.Stdout
	} else {
		if out, err = os.OpenFile(argOut, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
			return
		}
		defer out.Close()
	}

	var lis net.Listener

	if strings.Contains(argListen, ":") {
		if lis, err = net.Listen("tcp", argListen); err != nil {
			return
		}
	} else {
		if err = os.RemoveAll(argListen); err != nil {
			return
		}
		if lis, err = net.Listen("unix", argListen); err != nil {
			return
		}
	}

	defer lis.Close()

	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)

	var (
		doneServe = make(chan struct{})
		doneLines = make(chan struct{})
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lines := make(chan []byte)

	go func() {
		defer close(doneLines)
		for line := range lines {
			if _, err := out.Write(line); err != nil {
				return
			}
		}
		out.Sync()
	}()

	go func() {
		defer close(doneServe)
		chErr <- serveListener(ctx, lis, lines)
	}()

	select {
	case err = <-chErr:
		log.Println("error:", err.Error())
	case sig := <-chSig:
		log.Println("signal received:", sig.String())
		time.Sleep(time.Second * 3)
	}

	cancel()

	select {
	case <-doneServe:
	case <-time.After(time.Second * 3):
	}

	close(lines)

	<-doneLines
}

func serveListener(ctx context.Context, lis net.Listener, lines chan []byte) (err error) {
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			lis.Close()
		case <-done:
		}
	}()

	wg := &sync.WaitGroup{}

	for {
		var conn net.Conn
		if conn, err = lis.Accept(); err != nil {
			break
		}
		wg.Add(1)
		go handleConn(ctx, wg, conn, lines)
	}

	wg.Wait()
	return
}

func handleConn(ctx context.Context, wg *sync.WaitGroup, conn net.Conn, lines chan []byte) {
	defer wg.Done()
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	br := bufio.NewReader(conn)

	for {
		line, err := br.ReadBytes(Newline)

		if len(line) > 0 {
			if line[len(line)-1] != Newline {
				line = append(line, Newline)
			}
			lines <- line
		}

		if err != nil {
			return
		}
	}
}
