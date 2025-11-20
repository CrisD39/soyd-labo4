package datanode

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
	addr string
	node DataNode // <-- tu interfaz de almacenamiento
	ln   net.Listener
}

func NewTCPServer(addr string, node DataNode) Server {
	return &tcpServer{
		addr: addr, // ej: ":4001"
		node: node,
	}
}

func (s *tcpServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("datanode listen error on %s: %w", s.addr, err)
	}
	s.ln = ln
	fmt.Printf("Datanode listening on %s\n", s.addr)

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
			fmt.Printf("Datanode accept error: %v\n", err)
			return
		}
		go s.handleConn(conn)
	}
}

func (s *tcpServer) handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fmt.Printf("Datanode received: %q\n", line)

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		cmd := strings.ToUpper(fields[0])

		switch cmd {
		case "PING":
			// Comando súper simple para probar la conexión
			conn.Write([]byte("PONG\n"))

		case "WRITE":
			// Esperamos: WRITE <blockID> <size>
			if len(fields) != 3 {
				conn.Write([]byte("ERROR WRITE usage: WRITE <blockID> <size>\n"))
				continue
			}
			blockID := fields[1]
			sizeStr := fields[2]

			size, err := strconv.Atoi(sizeStr)
			if err != nil || size < 0 {
				conn.Write([]byte("ERROR invalid size\n"))
				continue
			}

			// Leer exactamente <size> bytes del bloque
			data := make([]byte, size)
			_, err = reader.Read(data)
			if err != nil {
				conn.Write([]byte("ERROR reading block data\n"))
				continue
			}

			// Guardar bloque usando tu DataNode
			if err := s.node.StoreBlock(blockID, data); err != nil {
				conn.Write([]byte("ERROR storing block\n"))
				continue
			}

			conn.Write([]byte("OK\n"))

		case "READ":
			// Esperamos: READ <blockID>
			if len(fields) != 2 {
				conn.Write([]byte("ERROR READ usage: READ <blockID>\n"))
				continue
			}
			blockID := fields[1]

			data, err := s.node.RetrieveBlock(blockID)
			if err != nil {
				conn.Write([]byte("ERROR block not found\n"))
				continue
			}

			// Enviar: DATA <size>\n<bytes>
			header := fmt.Sprintf("DATA %d\n", len(data))
			if _, err := conn.Write([]byte(header)); err != nil {
				return
			}
			if _, err := conn.Write(data); err != nil {
				return
			}

		default:
			conn.Write([]byte("ERROR unknown command\n"))
		}
	}
}
