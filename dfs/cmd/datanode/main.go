package main

import (
	"log"
	"os"

	"dfs/internal/config"
	"dfs/internal/datanode"
	dfslog "dfs/internal/logger"
)

func main() {
	
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("no se pudo cargar config.json: %v", err)
	}

	var nodeID string

	if len(os.Args) > 1 {
		nodeID = os.Args[1]
	} else if envID := os.Getenv("DATANODE_ID"); envID != "" {
		nodeID = envID
	} else if len(cfg.Datanodes) > 0 {
		nodeID = cfg.Datanodes[0].ID
		log.Printf("DATANODE_ID no especificado, usando primero del config.json: %s", nodeID)
	} else {
		log.Fatal("no hay datanodes definidos en config.json")
	}

	var dnCfgFound bool
	var dnCfg struct {
		ID          string
		ListenAddr  string
		LoggerAddr  string
		StoragePath string
	}

	for _, dn := range cfg.Datanodes {
		if dn.ID == nodeID {
			dnCfg = struct {
				ID          string
				ListenAddr  string
				LoggerAddr  string
				StoragePath string
			}{
				ID:          dn.ID,
				ListenAddr:  dn.ListenAddr,
				LoggerAddr:  dn.LoggerAddr,
				StoragePath: dn.StoragePath,
			}
			dnCfgFound = true
			break
		}
	}

	if !dnCfgFound {
		log.Fatalf("no se encontró configuración para el datanode con id=%s en config.json", nodeID)
	}

	
	componentName := "datanode-" + dnCfg.ID
	if err := dfslog.Init(componentName, dnCfg.LoggerAddr); err != nil {
		log.Printf("dfslog: no se pudo conectar al log server (%s): %v", dnCfg.LoggerAddr, err)
	}
	dfslog.Infof("DataNode %s iniciando. listen=%s storage=%s logger=%s",
		dnCfg.ID, dnCfg.ListenAddr, dnCfg.StoragePath, dnCfg.LoggerAddr)

	
	dn, err := datanode.NewFileDataNode(dnCfg.StoragePath)
	if err != nil {
		dfslog.Errorf("error creando fileDataNode en %s: %v", dnCfg.StoragePath, err)
		log.Fatal(err)
	}

	
	srv := datanode.NewTCPServer(dnCfg.ListenAddr, dn)
	if err := srv.Start(); err != nil {
		dfslog.Errorf("error iniciando servidor TCP del DataNode en %s: %v", dnCfg.ListenAddr, err)
		log.Fatal(err)
	}

	dfslog.Infof("DataNode %s escuchando en %s y listo para recibir WRITE/READ",
		dnCfg.ID, dnCfg.ListenAddr)

	
	select {}
}
