//go:build e2e

package e2e_test

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type fakeNntpServer struct {
	listener     net.Listener
	port         int
	mu           sync.Mutex
	articleCount int
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

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return // connection closed
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		if cmd == "" {
			continue
		}
		switch {
		case strings.HasPrefix(cmd, "AUTHINFO USER"):
			fmt.Fprintf(conn, "381 Enter password\r\n")
		case strings.HasPrefix(cmd, "AUTHINFO PASS"):
			fmt.Fprintf(conn, "281 Authentication accepted\r\n")
		case cmd == "CAPABILITIES":
			fmt.Fprintf(conn, "101 Capability list:\r\nVERSION 2\r\nREADER\r\nPOST\r\nDATE\r\n.\r\n")
		case cmd == "DATE":
			// nntppool sends DATE as its connectivity ping (RFC 3977 §7.1)
			fmt.Fprintf(conn, "111 %s\r\n", time.Now().UTC().Format("20060102150405"))
		case cmd == "POST":
			// Phase 1: accept article
			fmt.Fprintf(conn, "340 Send article\r\n")
			// Phase 2: read article body until lone ".\r\n"
			for {
				bodyLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(bodyLine, "\r\n") == "." {
					break
				}
			}
			// Extract Message-ID from the article for STAT verification
			s.mu.Lock()
			s.articleCount++
			s.mu.Unlock()
			fmt.Fprintf(conn, "240 Article posted\r\n")
		case strings.HasPrefix(cmd, "STAT "):
			// Post-check: always report article exists
			msgID := strings.TrimSpace(line[5:])
			fmt.Fprintf(conn, "223 0 %s\r\n", msgID)
		case cmd == "QUIT":
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
