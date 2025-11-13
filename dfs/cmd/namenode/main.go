package main

import (
	"log"

	"dfs/internal/metadata"
	"dfs/internal/namenode"
)

func main() {
	// creo el metadata store
	store := metadata.NewJSONStore("archivoPrueba.json")

	if err := store.Load(); err != nil {
		log.Fatalf("error loading metadata: %v", err)
	}
	log.Println("Metadata loaded successfully.")

	// crear el servicio del Namenode
	svc := namenode.NewService(store)

	// crear comunicación tcp
	server := namenode.NewTCPServer(":3000", svc)

	// comenzar el server
	if err := server.Start(); err != nil {
		log.Fatalf("error starting namenode server: %v", err)
	}

	log.Println("Namenode server running on :3000")

	// con esto lo mantengo corriendo
	select {}
}
