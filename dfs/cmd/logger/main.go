package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type LogServer struct {
	mu      sync.Mutex // protege el archivo
	file    *os.File
	entries chan string // cola de mensajes
}

func NewLogServer(path string) (*LogServer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	s := &LogServer{
		file:    f,
		entries: make(chan string, 1000), // cola con buffer
	}

	// Goroutine que vacía la cola y escribe al archivo
	go s.loopWriter()

	return s, nil
}

func (s *LogServer) loopWriter() {
	for line := range s.entries {
		s.mu.Lock()
		// Escribimos una línea por log (ya viene sin \n, lo agregamos acá)
		fmt.Fprintln(s.file, line)
		s.mu.Unlock()
	}
}

func (s *LogServer) handleConn(conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		msg := scanner.Text()

		// Le agregamos timestamp y origen acá (podrías hacerlo en el cliente también)
		t := time.Now().Format(time.RFC3339Nano)
		line := fmt.Sprintf("%s [%s] %s", t, remote, msg)

		// Encolamos; si la cola se llena, esto se bloquea hasta que el writer vacíe
		s.entries <- line
	}

	if err := scanner.Err(); err != nil {
		log.Printf("error leyendo de %s: %v\n", remote, err)
	}
}

func main() {
	// Podés hacer que lea de env o flags, acá lo dejo fijo para arrancar
	const addr = ":5000"
	const logPath = "dfs.log"

	server, err := NewLogServer(logPath)
	if err != nil {
		log.Fatalf("no pude crear LogServer: %v", err)
	}
	defer server.file.Close()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("no pude escuchar en %s: %v", addr, err)
	}
	log.Printf("LogServer escuchando en %s, escribiendo en %s\n", addr, logPath)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("error en Accept: %v\n", err)
			continue
		}
		go server.handleConn(conn)
	}
}
