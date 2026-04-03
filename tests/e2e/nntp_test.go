//go:build e2e

package e2e_test

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type fakeNntpServer struct {
	listener net.Listener
	port     int
}

func startFakeNntpServer() (*fakeNntpServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("fake NNTP listen: %w", err)
	}
	srv := &fakeNntpServer{
		listener: ln,
		port:     ln.Addr().(*net.TCPAddr).Port,
	}
	go srv.serve()
	return srv, nil
}

func (s *fakeNntpServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go s.handleConn(conn)
	}
}

func (s *fakeNntpServer) handleConn(conn net.Conn) {
	defer conn.Close()
	// RFC 3977 §5.1 greeting
	fmt.Fprintf(conn, "200 Postie-test NNTP server ready\r\n")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := strings.ToUpper(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "AUTHINFO USER"):
			fmt.Fprintf(conn, "381 Enter password\r\n")
		case strings.HasPrefix(line, "AUTHINFO PASS"):
			fmt.Fprintf(conn, "281 Authentication accepted\r\n")
		case line == "CAPABILITIES":
			fmt.Fprintf(conn, "101 Capability list:\r\nVERSION 2\r\nREADER\r\nPOST\r\nDATE\r\n.\r\n")
		case line == "DATE":
			// nntppool sends DATE as its connectivity ping (RFC 3977 §7.1)
			fmt.Fprintf(conn, "111 %s\r\n", time.Now().UTC().Format("20060102150405"))
		case line == "QUIT":
			fmt.Fprintf(conn, "205 closing connection\r\n")
			return
		default:
			fmt.Fprintf(conn, "500 Unknown command\r\n")
		}
	}
}

func (s *fakeNntpServer) Close() {
	s.listener.Close()
}
