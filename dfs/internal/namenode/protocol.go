package namenode

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Server interface {
	Start() error
	Stop() error
}

type tcpServer struct {
	addr    string
	service Service
	ln      net.Listener
}


func NewTCPServer(addr string, svc Service) Server {
	return &tcpServer{
		addr:    addr, // ej: ":3000"
		service: svc,
	}
}

func (s *tcpServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("tcp listen error on %s: %w", s.addr, err)
	}
	s.ln = ln
	fmt.Printf("Namenode listening on %s\n", s.addr)

	go s.acceptLoop()
	return nil
}

func (s *tcpServer) Stop() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *tcpServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %v\n", err)
			return
		}
		go s.handleConn(conn)
	}
}

func (s *tcpServer) handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		//leer hasta un enter
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fmt.Printf("Received message: %q\n", line)

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		cmd := strings.ToUpper(fields[0])

		switch cmd {
		case "PUT":
			if len(fields) != 3 {
				conn.Write([]byte("ERROR PUT usage: PUT <filename> <sizeBytes>\n"))
				continue
			}
			filename := fields[1]
			sizeStr := fields[2]

			sizeBytes, err := strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				conn.Write([]byte("ERROR invalid size\n"))
				continue
			}

			locations, err := s.service.HandlePut(filename, sizeBytes)
			if err != nil {
				conn.Write([]byte("ERROR HandlePut failed\n"))
				continue
			}

			var parts []string
			for _, loc := range locations {
				parts = append(parts, fmt.Sprintf("%s@%s", loc.BlockID, loc.Node))
			}
			conn.Write([]byte("METADATA " + strings.Join(parts, " ") + "\n"))

		case "GET":
			if len(fields) != 2 {
				conn.Write([]byte("ERROR GET usage: GET <filename>\n"))
				continue
			}
			filename := fields[1]

			locations, exists, err := s.service.HandleGet(filename)
			if err != nil {
				conn.Write([]byte("ERROR retrieving metadata\n"))
				continue
			}
			if !exists {
				conn.Write([]byte("NOT_FOUND\n"))
				continue
			}

			var parts []string
			for _, loc := range locations {
				parts = append(parts, fmt.Sprintf("%s@%s", loc.BlockID, loc.Node))
			}
			conn.Write([]byte("METADATA " + strings.Join(parts, " ") + "\n"))

		case "LS":
			keys, err := s.service.HandleList()
			if err != nil {
				conn.Write([]byte("ERROR listing keys\n"))
				continue
			}
			conn.Write([]byte("KEYS " + strings.Join(keys, " ") + "\n"))

		case "INFO":
			if len(fields) != 2 {
				conn.Write([]byte("ERROR INFO usage: INFO <filename>\n"))
				continue
			}
			filename := fields[1]

			locations, exists, err := s.service.HandleInfo(filename)
			if err != nil {
				conn.Write([]byte("ERROR retrieving info\n"))
				continue
			}
			if !exists {
				conn.Write([]byte("NOT_FOUND\n"))
				continue
			}

			var parts []string
			for _, loc := range locations {
				parts = append(parts, fmt.Sprintf("%s@%s", loc.BlockID, loc.Node))
			}
			conn.Write([]byte("METADATA " + strings.Join(parts, " ") + "\n"))

		default:
			conn.Write([]byte("ERROR unknown command\n"))
		}
	}
}
