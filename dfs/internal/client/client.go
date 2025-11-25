package client

// client.go client TCP
import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	dfslog "dfs/internal/logger" // 👈 usamos el mismo logger que en namenode/datanode
)

type PutPlan struct {
	FileName  string      `json:"file_name"`
	BlockSize int         `json:"block_size"`
	Blocks    []BlockPlan `json:"blocks"`
	Status    string      `json:"status"`
}

type BlockPlan struct {
	BlockID  string   `json:"block_id"`
	Replicas []string `json:"replicas"`
}

type Client interface {
	Put(localPath string, remoteName string) error
	Get(remoteName string, localPath string) error
	Info(remoteName string) error
	List() ([]string, error)
	Ping() (string, error)
	Delete(remoteName string) error
}


type tcpClient struct {
	addr string
}

func NewTCPClient(addr string) Client {
	return &tcpClient{addr: addr}
}

func (c *tcpClient) sendCommand(line string) (string, error) {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "%s\n", line); err != nil {
		return "", err
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}

	response = strings.TrimSpace(response)
	return response, nil
}

func (c *tcpClient) Ping() (string, error) {
	resp, err := c.sendCommand("PING")
	if err != nil {
		return "", fmt.Errorf("error enviando PING al servidor (%s): %w", c.addr, err)
	}
	dfslog.Infof("Respuesta PING del namenode (%s): %s", c.addr, resp)
	return resp, nil
}

func (c *tcpClient) Put(localPath string, remoteName string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("error leyendo archivo local %q: %w", localPath, err)
	}

	sizeBytes := len(data)
	dfslog.Infof("Iniciando PUT local=%s remote=%s size=%d bytes", localPath, remoteName, sizeBytes)

	cmd := fmt.Sprintf("PUT %s %d", remoteName, sizeBytes)

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return fmt.Errorf("error enviando PUT al namenode: %w", err)
	}

	if strings.HasPrefix(resp, "ERROR") {
		return fmt.Errorf("namenode respondió error: %s", resp)
	}

	if !strings.HasPrefix(resp, "METADATA ") {
		return fmt.Errorf("respuesta inesperada del namenode (se esperaba METADATA): %s", resp)
	}

	metaJSON := strings.TrimPrefix(resp, "METADATA ")

	var plan PutPlan
	if err := json.Unmarshal([]byte(metaJSON), &plan); err != nil {
		return fmt.Errorf("error parseando METADATA del namenode: %w", err)
	}

	if plan.Status != "" && strings.ToUpper(plan.Status) != "OK" {
		return fmt.Errorf("namenode devolvió estado no OK para PUT: %s", plan.Status)
	}

	if plan.BlockSize <= 0 {
		return fmt.Errorf("plan de PUT inválido: block_size <= 0")
	}

	blockSize := plan.BlockSize
	dfslog.Infof("Plan de PUT recibido: file=%s block_size=%d blocks=%d",
		plan.FileName, blockSize, len(plan.Blocks))

	for i, b := range plan.Blocks {
		start := i * blockSize
		end := start + blockSize

		if start >= len(data) {
			return fmt.Errorf("plan inconsistente: bloque %d empieza fuera del archivo", i)
		}
		if end > len(data) {
			end = len(data)
		}

		blockData := data[start:end]

		dfslog.Debugf("Subiendo bloque %d (%s) rango=[%d:%d] replicas=%v",
			i, b.BlockID, start, end, b.Replicas)

		if err := c.uploadBlock(b, blockData); err != nil {
			return fmt.Errorf("error subiendo bloque %d (%s): %w", i, b.BlockID, err)
		}
	}

	dfslog.Infof("PUT completado: archivo %s subido en %d bloques con replicación",
		remoteName, len(plan.Blocks))

	return nil
}

func (c *tcpClient) Delete(remoteName string) error {
	cmd := fmt.Sprintf("DELETE %s", remoteName)

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return fmt.Errorf("error enviando DELETE al namenode: %w", err)
	}

	if resp == "NOT_FOUND" {
		return fmt.Errorf("archivo no encontrado en namenode: %s", remoteName)
	}
	if strings.HasPrefix(resp, "ERROR") {
		return fmt.Errorf("namenode respondió error: %s", resp)
	}
	if resp != "OK" {
		return fmt.Errorf("respuesta inesperada del namenode a DELETE: %s", resp)
	}

	dfslog.Infof("DELETE completado: archivo %s eliminado en namenode", remoteName)
	return nil
}


func (c *tcpClient) uploadBlock(b BlockPlan, data []byte) error {
	if len(b.Replicas) == 0 {
		return fmt.Errorf("bloque %s no tiene réplicas asignadas", b.BlockID)
	}

	for _, addr := range b.Replicas {
		dfslog.Debugf("Enviando bloque %s a datanode %s (size=%d)", b.BlockID, addr, len(data))
		if err := storeToDataNode(addr, b.BlockID, data); err != nil {
			return fmt.Errorf("error guardando bloque en datanode %s: %w", addr, err)
		}
	}

	return nil
}

func storeToDataNode(addr, blockID string, data []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("no se pudo conectar al datanode %s: %w", addr, err)
	}
	defer conn.Close()

	header := fmt.Sprintf("WRITE %s %d\n", blockID, len(data))
	if _, err := conn.Write([]byte(header)); err != nil {
		return fmt.Errorf("error enviando header al datanode %s: %w", addr, err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("error enviando datos del bloque al datanode %s: %w", addr, err)
	}

	ack, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("error leyendo ACK del datanode %s: %w", addr, err)
	}

	if strings.TrimSpace(ack) != "OK" {
		return fmt.Errorf("datanode %s respondió error: %s", addr, strings.TrimSpace(ack))
	}

	return nil
}

func (c *tcpClient) Get(remoteName string, localPath string) error {
	cmd := fmt.Sprintf("GET %s", remoteName)

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return fmt.Errorf("error enviando GET al namenode: %w", err)
	}

	if resp == "NOT_FOUND" {
		return fmt.Errorf("archivo no encontrado en namenode: %s", remoteName)
	}
	if strings.HasPrefix(resp, "ERROR") {
		return fmt.Errorf("namenode respondió error: %s", resp)
	}

	if !strings.HasPrefix(resp, "METADATA ") {
		return fmt.Errorf("respuesta inesperada del namenode (se esperaba METADATA): %s", resp)
	}

	metaJSON := strings.TrimPrefix(resp, "METADATA ")
	dfslog.Debugf("METADATA recibido en GET para %s: %s", remoteName, metaJSON)

	var plan PutPlan
	if err := json.Unmarshal([]byte(metaJSON), &plan); err != nil {
		return fmt.Errorf("error parseando METADATA del namenode: %w", err)
	}

	if len(plan.Blocks) == 0 {
		return fmt.Errorf("el archivo %s no tiene bloques registrados", remoteName)
	}

	dfslog.Infof("Plan de GET: file=%s blocks=%d", plan.FileName, len(plan.Blocks))

	var buf bytes.Buffer

	for _, b := range plan.Blocks {
		if len(b.Replicas) == 0 {
			return fmt.Errorf("bloque %s no tiene réplicas asignadas", b.BlockID)
		}

		var blockData []byte
		var lastErr error

		for _, addr := range b.Replicas {
			dfslog.Debugf("Intentando leer bloque %s desde %s", b.BlockID, addr)
			blockData, lastErr = readFromDataNode(addr, b.BlockID)
			if lastErr == nil {
				dfslog.Debugf("Bloque %s leído correctamente desde %s (size=%d)", b.BlockID, addr, len(blockData))
				break
			}
			dfslog.Infof("WARN: fallo leer bloque %s desde %s: %v", b.BlockID, addr, lastErr)
		}

		if lastErr != nil {
			return fmt.Errorf("no se pudo leer el bloque %s de ninguna réplica: %w", b.BlockID, lastErr)
		}

		if _, err := buf.Write(blockData); err != nil {
			return fmt.Errorf("error acumulando datos del bloque %s: %w", b.BlockID, err)
		}
	}

	if err := os.WriteFile(localPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("error escribiendo archivo local %q: %w", localPath, err)
	}

	dfslog.Infof("GET completado: archivo %s reconstruido en %s (%d bytes)",
		remoteName, localPath, buf.Len())

	return nil
}

func readFromDataNode(addr, blockID string) ([]byte, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar al datanode %s: %w", addr, err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "READ %s\n", blockID); err != nil {
		return nil, fmt.Errorf("error enviando READ al datanode %s: %w", addr, err)
	}

	reader := bufio.NewReader(conn)

	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error leyendo header DATA desde %s: %w", addr, err)
	}

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "DATA ") {
		return nil, fmt.Errorf("respuesta inesperada del datanode %s: %q", addr, header)
	}

	sizeStr := strings.TrimPrefix(header, "DATA ")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 0 {
		return nil, fmt.Errorf("tamaño inválido en header DATA desde %s: %q", addr, header)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("error leyendo %d bytes de bloque desde %s: %w", size, addr, err)
	}

	return data, nil
}

func (c *tcpClient) Info(remoteName string) error {
	cmd := fmt.Sprintf("INFO %s", remoteName)

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return fmt.Errorf("error enviando INFO al namenode: %w", err)
	}

	if resp == "NOT_FOUND" {
		return fmt.Errorf("archivo no encontrado en namenode: %s", remoteName)
	}
	if strings.HasPrefix(resp, "ERROR") {
		return fmt.Errorf("namenode respondió error: %s", resp)
	}

	dfslog.Infof("Respuesta INFO para %s: %s", remoteName, resp)

	return nil
}

func (c *tcpClient) List() ([]string, error) {
	resp, err := c.sendCommand("LS")
	if err != nil {
		return nil, fmt.Errorf("error enviando LS al namenode: %w", err)
	}

	if strings.HasPrefix(resp, "ERROR") {
		return nil, fmt.Errorf("namenode respondió error: %s", resp)
	}

	parts := strings.Fields(resp)
	if len(parts) == 0 {
		return nil, fmt.Errorf("respuesta vacía del namenode")
	}

	if parts[0] != "KEYS" {
		return nil, fmt.Errorf("respuesta inesperada de LS: %s", resp)
	}

	if len(parts) == 1 {
		dfslog.Infof("LS: el namenode no tiene archivos registrados")
		return []string{}, nil
	}

	files := parts[1:]
	dfslog.Infof("LS: el namenode devolvió %d archivos", len(files))
	return files, nil
}
