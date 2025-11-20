package main

import (
	"fmt"
	"log"
	"os"

	"dfs/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso:")
		fmt.Println("  client ls")
		fmt.Println("  client info <remoteName>")
		fmt.Println("  client put <localPath> <remoteName>")
		fmt.Println("  client get <remoteName> <localPath>")
		return
	}

	c := client.NewTCPClient("localhost:3000")

	switch os.Args[1] {
	case "ls":
		files, err := c.List()
		if err != nil {
			log.Fatal(err)
		}
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
		if err := c.Info(os.Args[2]); err != nil {
			log.Fatal(err)
		}

	case "put":
		if len(os.Args) != 4 {
			log.Fatal("uso: client put <localPath> <remoteName>")
		}
		if err := c.Put(os.Args[2], os.Args[3]); err != nil {
			log.Fatal(err)
		}

	case "get":
		if len(os.Args) != 4 {
			log.Fatal("uso: client get <remoteName> <localPath>")
		}
		if err := c.Get(os.Args[2], os.Args[3]); err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatalf("comando desconocido: %s", os.Args[1])
	}
}
