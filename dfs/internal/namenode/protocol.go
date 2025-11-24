package namenode

// protocol.go namenode TCP
import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	dfslog "dfs/internal/logger" // 👈 nuevo import
)

// Respuesta que el namenode le envía al cliente cuando hace un PUT.
// Es lo que después va a parsear el cliente.
type PutBlock struct {
	BlockID  string   `json:"block_id"`
	Replicas []string `json:"replicas"`
}

type PutResponse struct {
	FileName  string     `json:"file_name"`
	BlockSize int64      `json:"block_size"`
	Blocks    []PutBlock `json:"blocks"`
	Status    string     `json:"status"` // <--- nuevo
}

type Server interface {
	Start() error
	Stop() error
}

type tcpServer struct {
	addr    string
	service Namenode
	ln      net.Listener
}

func NewTCPServer(addr string, svc Namenode) Server {
	return &tcpServer{
		addr:    addr,
		service: svc,
	}
}

func (s *tcpServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("tcp listen error on %s: %w", s.addr, err)
	}
	s.ln = ln
	dfslog.Infof("Namenode TCP escuchando en %s", s.addr)

	go s.acceptLoop()
	return nil
}

func (s *tcpServer) Stop() error {
	if s.ln != nil {
		dfslog.Infof("Namenode TCP deteniéndose en %s", s.addr)
		return s.ln.Close()
	}
	return nil
}

func (s *tcpServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			// si se cerró el listener, lo registramos y salimos
			dfslog.Errorf("TCP accept error: %v", err)
			return
		}
		dfslog.Infof("Nueva conexión desde %s", conn.RemoteAddr().String())
		go s.handleConn(conn)
	}
}

func (s *tcpServer) handleConn(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	dfslog.Debugf("handleConn iniciado para %s", remote)

	reader := bufio.NewReader(conn)

	for {
		// leer hasta un enter
		line, err := reader.ReadString('\n')
		if err != nil {
			// normalmente va a ser EOF cuando el cliente cierra
			dfslog.Debugf("Conexión cerrada desde %s: %v", remote, err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		dfslog.Debugf("Mensaje recibido de %s: %q", remote, line)

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		cmd := strings.ToUpper(fields[0])

		switch cmd {
		case "PING":
			dfslog.Debugf("PING recibido desde %s", remote)
			conn.Write([]byte("PONG\n"))

		case "PUT":
			if len(fields) != 3 {
				dfslog.Errorf("PUT inválido desde %s: %q", remote, line)
				conn.Write([]byte("ERROR PUT usage: PUT <filename> <sizeBytes>\n"))
				continue
			}
			filename := fields[1]
			sizeStr := fields[2]

			sizeBytes, err := strconv.ParseInt(sizeStr, 10, 64)
			if err != nil {
				dfslog.Errorf("PUT tamaño inválido desde %s: %q (%v)", remote, line, err)
				conn.Write([]byte("ERROR invalid size\n"))
				continue
			}

			dfslog.Infof("HandlePut TCP: filename=%s size=%d desde %s", filename, sizeBytes, remote)
			locations, err := s.service.HandlePut(filename, sizeBytes)
			if err != nil {
				dfslog.Errorf("HandlePut falló para %s: %v", filename, err)
				conn.Write([]byte("ERROR HandlePut failed\n"))
				continue
			}

			// Armamos el JSON METADATA para el cliente.
			blocks := make([]PutBlock, len(locations))
			for i, loc := range locations {
				blocks[i] = PutBlock{
					BlockID:  loc.BlockID,
					Replicas: append([]string(nil), loc.Replicas...), // futuro: varios nodos
				}
			}

			resp := PutResponse{
				FileName:  filename,
				BlockSize: BlockSize, // constante definida en namenode.go
				Blocks:    blocks,
				Status:    "OK",
			}

			data, err := json.Marshal(resp)
			if err != nil {
				dfslog.Errorf("Error encodeando METADATA para %s: %v", filename, err)
				conn.Write([]byte("ERROR encoding METADATA\n"))
				continue
			}

			dfslog.Debugf("Enviando METADATA PUT para %s a %s", filename, remote)
			conn.Write([]byte("METADATA " + string(data) + "\n"))

		case "GET":
			if len(fields) != 2 {
				dfslog.Errorf("GET inválido desde %s: %q", remote, line)
				conn.Write([]byte("ERROR GET usage: GET <filename>\n"))
				continue
			}

			filename := fields[1]
			dfslog.Infof("HandleGet TCP: filename=%s desde %s", filename, remote)

			locations, exists, err := s.service.HandleGet(filename)
			if err != nil {
				dfslog.Errorf("HandleGet falló para %s: %v", filename, err)
				conn.Write([]byte("ERROR retrieving metadata\n"))
				continue
			}
			if !exists {
				dfslog.Infof("GET: archivo no encontrado %s (desde %s)", filename, remote)
				conn.Write([]byte("NOT_FOUND\n"))
				continue
			}

			// Armamos el mismo JSON que en PUT
			resp := PutResponse{
				FileName:  filename,
				BlockSize: BlockSize, // misma constante que usás en PUT
				Blocks:    make([]PutBlock, len(locations)),
				Status:    "OK",
			}

			for i, loc := range locations {
				resp.Blocks[i] = PutBlock{
					BlockID:  loc.BlockID,
					Replicas: loc.Replicas,
				}
			}

			data, err := json.Marshal(resp)
			if err != nil {
				dfslog.Errorf("Error encodeando METADATA en GET para %s: %v", filename, err)
				conn.Write([]byte("ERROR encoding METADATA\n"))
				continue
			}

			dfslog.Debugf("Enviando METADATA GET para %s a %s", filename, remote)
			conn.Write([]byte("METADATA " + string(data) + "\n"))

		case "LS":
			dfslog.Infof("LS recibido desde %s", remote)

			keys, err := s.service.HandleList()
			if err != nil {
				dfslog.Errorf("HandleList falló: %v", err)
				conn.Write([]byte("ERROR listing keys\n"))
				continue
			}
			conn.Write([]byte("KEYS " + strings.Join(keys, " ") + "\n"))

		case "INFO":
			if len(fields) != 2 {
				dfslog.Errorf("INFO inválido desde %s: %q", remote, line)
				conn.Write([]byte("ERROR INFO usage: INFO <filename>\n"))
				continue
			}
			filename := fields[1]

			dfslog.Infof("HandleInfo TCP: filename=%s desde %s", filename, remote)
			locations, exists, err := s.service.HandleInfo(filename)
			if err != nil {
				dfslog.Errorf("HandleInfo falló para %s: %v", filename, err)
				conn.Write([]byte("ERROR retrieving info\n"))
				continue
			}
			if !exists {
				dfslog.Infof("INFO: archivo no encontrado %s (desde %s)", filename, remote)
				conn.Write([]byte("NOT_FOUND\n"))
				continue
			}

			var parts []string
			for _, loc := range locations {
				parts = append(parts, fmt.Sprintf("%s@%s", loc.BlockID, loc.Replicas))
			}
			conn.Write([]byte("METADATA " + strings.Join(parts, " ") + "\n"))

		default:
			dfslog.Errorf("Comando desconocido desde %s: %q", remote, line)
			conn.Write([]byte("ERROR unknown command\n"))
		}
	}
}
