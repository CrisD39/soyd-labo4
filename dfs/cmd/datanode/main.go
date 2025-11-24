package main

import (
	"log"

	"dfs/internal/datanode"
	"dfs/internal/logger" // 👈 logger centralizado
)

func main() {
	// Inicializar logger remoto (log server en localhost:5000)
	if err := dfslog.Init("datanode-4001", "localhost:5000"); err != nil {
		// Si falla el logger, al menos avisamos por stderr
		log.Printf("dfslog: no se pudo conectar al log server: %v", err)
	}
	dfslog.Infof("DataNode iniciando en :4002")

	// Crear DataNode de archivos (guarda bloques en ./data)
	dn, err := datanode.NewFileDataNode("./data") // carpeta donde guardar bloques
	if err != nil {
		dfslog.Errorf("error creando fileDataNode: %v", err)
		log.Fatal(err)
	}

	// Levantar servidor TCP en :4002
	srv := datanode.NewTCPServer(":4001", dn)
	if err := srv.Start(); err != nil {
		dfslog.Errorf("error iniciando servidor TCP del DataNode: %v", err)
		log.Fatal(err)
	}

	dfslog.Infof("DataNode escuchando en :4002 y listo para recibir WRITE/READ")
	select {}
}
