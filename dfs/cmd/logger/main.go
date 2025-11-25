package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"dfs/internal/config"
)

type LogServer struct {
	mu      sync.Mutex
	file    *os.File
	entries chan string
}

func NewLogServer(path string) (*LogServer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	s := &LogServer{
		file:    f,
		entries: make(chan string, 1000),
	}

	go s.loopWriter()

	return s, nil
}

func (s *LogServer) loopWriter() {
	for line := range s.entries {
		s.mu.Lock()
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
		t := time.Now().Format(time.RFC3339Nano)
		line := fmt.Sprintf("%s [%s] %s", t, remote, msg)
		s.entries <- line
	}

	if err := scanner.Err(); err != nil {
		log.Printf("error leyendo de %s: %v\n", remote, err)
	}
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("no pude cargar config.json: %v", err)
	}

	addr := cfg.Logger.ListenAddr
	if addr == "" {
		addr = ":5000"
	}

	logPath := "dfs.log"

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
