package dfslog

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type logEntry struct {
	level string
	msg   string
}

var (
	component string
	addr      string

	conn     net.Conn
	entries  chan logEntry
	initOnce sync.Once
	muConn   sync.Mutex
)

// Init establece el nombre del componente y se conecta al logserver.
// Ej: dfslog.Init("namenode", "localhost:4000")
func Init(comp, serverAddr string) error {
	var err error
	initOnce.Do(func() {
		component = comp
		addr = serverAddr
		entries = make(chan logEntry, 1000)

		// Intentamos una única conexión persistente
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			// si falla, dejamos conn=nil y hacemos fallback a stderr en el writer
			fmt.Fprintf(os.Stderr, "dfslog: no se pudo conectar a %s: %v\n", addr, err)
		}

		// Goroutine que consume la cola y escribe al socket
		go writerLoop()
	})
	return err
}

func writerLoop() {
	for e := range entries {
		t := time.Now().Format(time.RFC3339Nano)
		line := fmt.Sprintf("%s [%s] [%s] %s\n", t, component, e.level, e.msg)

		muConn.Lock()
		if conn != nil {
			_, err := conn.Write([]byte(line))
			if err != nil {
				// si se rompe la conexión, hacemos fallback a stderr
				fmt.Fprintf(os.Stderr, "dfslog: error escribiendo al logserver: %v\n", err)
				_ = conn.Close()
				conn = nil
			}
		}
		if conn == nil {
			// fallback local
			fmt.Fprint(os.Stderr, line)
		}
		muConn.Unlock()
	}
}

func logf(level, format string, args ...interface{}) {
	if entries == nil {
		// Init nunca se llamó; logueamos algo mínimo a stderr
		line := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "[%s] %s\n", level, line)
		return
	}
	msg := fmt.Sprintf(format, args...)
	entries <- logEntry{level: level, msg: msg}
}

func Infof(format string, args ...interface{})  { logf("INFO", format, args...) }
func Errorf(format string, args ...interface{}) { logf("ERROR", format, args...) }
func Debugf(format string, args ...interface{}) { logf("DEBUG", format, args...) }
