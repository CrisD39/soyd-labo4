package datanode

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	dfslog "dfs/internal/logger" 
)

type Server interface {
	Start() error
	Stop() error
}

type tcpServer struct {
	addr string
	node DataNode
	ln   net.Listener
}

func NewTCPServer(addr string, node DataNode) Server {
	return &tcpServer{
		addr: addr, 
		node: node,
	}
}

func (s *tcpServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("datanode listen error on %s: %w", s.addr, err)
	}
	s.ln = ln
	dfslog.Infof("Datanode escuchando en %s", s.addr)

	go s.acceptLoop()
	return nil
}

func (s *tcpServer) Stop() error {
	if s.ln != nil {
		dfslog.Infof("Datanode deteniéndose en %s", s.addr)
		return s.ln.Close()
	}
	return nil
}

func (s *tcpServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			dfslog.Errorf("Datanode accept error: %v", err)
			return
		}
		dfslog.Infof("Nueva conexión al Datanode desde %s", conn.RemoteAddr().String())
		go s.handleConn(conn)
	}
}

func (s *tcpServer) handleConn(conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	dfslog.Debugf("handleConn iniciado para %s", remote)

	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				dfslog.Errorf("Error leyendo comando desde %s: %v", remote, err)
			} else {
				dfslog.Debugf("Conexión cerrada desde %s", remote)
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		dfslog.Debugf("Datanode recibió de %s: %q", remote, line)

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		cmd := strings.ToUpper(fields[0])

		switch cmd {
		case "PING":
			dfslog.Debugf("PING recibido desde %s", remote)
			conn.Write([]byte("PONG\n"))

		case "WRITE":
			if len(fields) != 3 {
				dfslog.Errorf("WRITE inválido desde %s: %q", remote, line)
				conn.Write([]byte("ERROR WRITE usage: WRITE <blockID> <size>\n"))
				continue
			}
			blockID := fields[1]
			sizeStr := fields[2]

			size, err := strconv.Atoi(sizeStr)
			if err != nil || size < 0 {
				dfslog.Errorf("WRITE tamaño inválido desde %s: %q", remote, line)
				conn.Write([]byte("ERROR invalid size\n"))
				continue
			}

			dfslog.Infof("WRITE bloque=%s size=%d desde %s", blockID, size, remote)

		
			data := make([]byte, size)
			if _, err := io.ReadFull(reader, data); err != nil {
				dfslog.Errorf("Error leyendo datos del bloque %s desde %s: %v", blockID, remote, err)
				conn.Write([]byte("ERROR reading block data\n"))
				continue
			}

			if err := s.node.StoreBlock(blockID, data); err != nil {
				dfslog.Errorf("Error almacenando bloque %s: %v", blockID, err)
				conn.Write([]byte("ERROR storing block\n"))
				continue
			}

			dfslog.Infof("WRITE OK bloque=%s desde %s", blockID, remote)
			conn.Write([]byte("OK\n"))

		case "READ":
			if len(fields) != 2 {
				dfslog.Errorf("READ inválido desde %s: %q", remote, line)
				conn.Write([]byte("ERROR READ usage: READ <blockID>\n"))
				continue
			}
			blockID := fields[1]

			dfslog.Infof("READ bloque=%s desde %s", blockID, remote)

			data, err := s.node.RetrieveBlock(blockID)
			if err != nil {
				dfslog.Errorf("Bloque no encontrado %s (desde %s): %v", blockID, remote, err)
				conn.Write([]byte("ERROR block not found\n"))
				continue
			}

			header := fmt.Sprintf("DATA %d\n", len(data))
			if _, err := conn.Write([]byte(header)); err != nil {
				dfslog.Errorf("Error enviando header DATA para %s a %s: %v", blockID, remote, err)
				return
			}
			if _, err := conn.Write(data); err != nil {
				dfslog.Errorf("Error enviando datos del bloque %s a %s: %v", blockID, remote, err)
				return
			}
			dfslog.Debugf("READ OK bloque=%s a %s (size=%d)", blockID, remote, len(data))

		default:
			dfslog.Errorf("Comando desconocido desde %s: %q", remote, line)
			conn.Write([]byte("ERROR unknown command\n"))
		}
	}
}
