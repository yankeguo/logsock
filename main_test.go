package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	mathrand "math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testWriter struct {
	buf *bytes.Buffer
}

func (tw *testWriter) Write(p []byte) (int, error) {
	return tw.buf.Write(p)
}

func (tw *testWriter) Sync() error {
	return nil
}

func TestServeListener(t *testing.T) {
	buf := make([]byte, 1024)
	rand.Read(buf)
	line := []byte(hex.EncodeToString(buf) + "\n")
	batch := bytes.Repeat(line, 10)

	tw := &testWriter{buf: &bytes.Buffer{}}

	os.RemoveAll("test.sock")
	listener, err := net.Listen("unix", "test.sock")
	require.NoError(t, err)

	lines := make(chan []byte)

	go func() {
		for line := range lines {
			if _, err := tw.Write(line); err != nil {
				return
			}
		}
	}()

	go serveListener(context.Background(), listener, lines)

	time.Sleep(time.Second)

	wg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {

		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.Dial("unix", "test.sock")
			require.NoError(t, err)
			defer conn.Close()

			i := 0

			for {
				n := mathrand.Intn(128)

				if i+n < len(batch) {
					_, err := conn.Write(batch[i : i+n])
					require.NoError(t, err)
					i += n
				} else {
					_, err := conn.Write(batch[i:])
					require.NoError(t, err)
					break
				}
			}
		}()
	}

	wg.Wait()

	close(lines)

	br := bufio.NewReader(bytes.NewReader(tw.buf.Bytes()))

	for {
		rline, err := br.ReadBytes('\n')
		if err == io.EOF {
			err = nil
		}
		require.NoError(t, err)

		if len(rline) == 0 {
			break
		}

		require.Equal(t, line, rline)
	}
}
