package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

type Client interface {
	Put(localPath string, remoteName string) error
	Get(remoteName string, localPath string) error
	Info(remoteName string) error
	List() ([]string, error)
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

func (c *tcpClient) Put(localPath string, remoteName string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("error leyendo archivo local %q: %w", localPath, err)
	}

	sizeBytes := info.Size()
	cmd := fmt.Sprintf("PUT %s %d", remoteName, sizeBytes)

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return fmt.Errorf("error enviando PUT al namenode: %w", err)
	}

	if strings.HasPrefix(resp, "ERROR") {
		return fmt.Errorf("namenode respondió error: %s", resp)
	}

	fmt.Println("Respuesta PUT:", resp)
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

	// Ej: "METADATA file.txt_block_1@node_placeholder ..."
	fmt.Println("Respuesta GET:", resp)
	// Más adelante: usar esta metadata para descargar bloques desde datanodes y escribir en localPath.
	return nil
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

	fmt.Println("Respuesta INFO:", resp)
	// Más adelante: parsear METADATA y mostrar algo más lindo (cantidad de bloques, nodos, etc.).
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

	// Formato esperado: "KEYS a.txt b.txt c.txt"
	parts := strings.Fields(resp)
	if len(parts) == 0 {
		return nil, fmt.Errorf("respuesta vacía del namenode")
	}

	if parts[0] != "KEYS" {
		return nil, fmt.Errorf("respuesta inesperada de LS: %s", resp)
	}

	if len(parts) == 1 {
		// "KEYS" solo → ningún archivo
		return []string{}, nil
	}

	files := parts[1:]
	return files, nil
}
