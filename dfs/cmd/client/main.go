package main

import (
	"fmt"
	"log"
	"os"

	"dfs/internal/client"
	"dfs/internal/config"
	dfslog "dfs/internal/logger"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("no se pudo cargar config.json: %v", err)
	}

	if err := dfslog.Init("client", cfg.Client.LoggerAddr); err != nil {
		log.Printf("dfslog: no se pudo conectar al log server (%s): %v", cfg.Client.LoggerAddr, err)
	}
	dfslog.Infof("cliente iniciado. logger=%s, namenode=%s, args=%v",
		cfg.Client.LoggerAddr,
		cfg.Client.NamenodeHTTPAddr,
		os.Args,
	)

	if len(os.Args) < 2 {
		fmt.Println("Uso:")
		fmt.Println("  client ping")
		fmt.Println("  client ls")
		fmt.Println("  client info <remoteName>")
		fmt.Println("  client put <localPath> <remoteName>")
		fmt.Println("  client get <remoteName> <localPath>")
		fmt.Println("  client delete <remoteName>")

		return
	}

	c := client.NewTCPClient(cfg.Client.NamenodeTCPAddr)

	cmd := os.Args[1]
	dfslog.Infof("comando recibido: %s", cmd)

	switch cmd {
	case "ping":
		dfslog.Infof("ejecutando ping()")

		resp, err := c.Ping()
		if err != nil {
			dfslog.Errorf("error en ping: %v", err)
			log.Fatal(err)
		}

		fmt.Println("respuesta de ping:", resp)

	case "delete":
		if len(os.Args) != 3 {
			log.Fatal("uso: client delete <remoteName>")
		}
		remote := os.Args[2]
		dfslog.Infof("ejecutando delete(%s)", remote)

		if err := c.Delete(remote); err != nil {
			dfslog.Errorf("error en delete(%s): %v", remote, err)
			log.Fatal(err)
		}
		dfslog.Infof("delete completado para %s", remote)

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
