package main

import (
	"fmt"
	"log"
	"os"

	"dfs/internal/client"
	"dfs/internal/logger" // <- usamos el logger central
)

func main() {
	// Inicializar logger remoto (logserver en localhost:5000)
	if err := dfslog.Init("client", "localhost:5000"); err != nil {
		// Si falla el logger externo, al menos avisamos por stderr
		log.Printf("dfslog: no se pudo conectar al log server: %v", err)
	}
	dfslog.Infof("cliente iniciado con args: %v", os.Args)

	if len(os.Args) < 2 {
		// Esto es “UX” para el usuario, no log del sistema
		fmt.Println("Uso:")
		fmt.Println("  client ping")
		fmt.Println("  client ls")
		fmt.Println("  client info <remoteName>")
		fmt.Println("  client put <localPath> <remoteName>")
		fmt.Println("  client get <remoteName> <localPath>")
		return
	}

	c := client.NewTCPClient("localhost:3000")

	cmd := os.Args[1]
	dfslog.Infof("comando recibido: %s", cmd)

	switch cmd {
	case "ping":
		dfslog.Infof("ejecutando ping()")
		if err := c.Ping(); err != nil {
			dfslog.Errorf("error en ping: %v", err)
			log.Fatal(err)
		}

	case "ls":
		dfslog.Infof("ejecutando ls()")
		files, err := c.List()
		if err != nil {
			dfslog.Errorf("error en ls: %v", err)
			log.Fatal(err)
		}
		dfslog.Infof("ls devolvió %d archivos", len(files))

		if len(files) == 0 {
			fmt.Println("No hay archivos en el namenode.")
		} else {
			fmt.Println("Archivos en el namenode:")
			for _, f := range files {
				fmt.Println(" -", f)
			}
		}

	case "info":
		if len(os.Args) != 3 {
			log.Fatal("uso: client info <remoteName>")
		}
		remote := os.Args[2]
		dfslog.Infof("ejecutando info(%s)", remote)

		if err := c.Info(remote); err != nil {
			dfslog.Errorf("error en info(%s): %v", remote, err)
			log.Fatal(err)
		}

	case "put":
		if len(os.Args) != 4 {
			log.Fatal("uso: client put <localPath> <remoteName>")
		}
		local := os.Args[2]
		remote := os.Args[3]
		dfslog.Infof("ejecutando put(local=%s, remote=%s)", local, remote)

		if err := c.Put(local, remote); err != nil {
			dfslog.Errorf("error en put(local=%s, remote=%s): %v", local, remote, err)
			log.Fatal(err)
		}
		dfslog.Infof("put completado correctamente para %s -> %s", local, remote)

	case "get":
		if len(os.Args) != 4 {
			log.Fatal("uso: client get <remoteName> <localPath>")
		}
		remote := os.Args[2]
		local := os.Args[3]
		dfslog.Infof("ejecutando get(remote=%s, local=%s)", remote, local)

		if err := c.Get(remote, local); err != nil {
			dfslog.Errorf("error en get(remote=%s, local=%s): %v", remote, local, err)
			log.Fatal(err)
		}
		dfslog.Infof("get completado correctamente para %s -> %s", remote, local)

	default:
		dfslog.Errorf("comando desconocido: %s", cmd)
		log.Fatalf("comando desconocido: %s", cmd)
	}
}
