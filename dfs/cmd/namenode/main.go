package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"dfs/internal/config"
	dfslog "dfs/internal/logger"
	"dfs/internal/metadata"
	"dfs/internal/namenode"
)

type DataNodeConfig struct {
	ID   string `json:"id"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ClusterConfig struct {
	Nodes []DataNodeConfig `json:"nodes"`
}

func loadDataNodesFromFile(path string) ([]string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo %s: %w", path, err)
	}

	var cfg ClusterConfig
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.Nodes) == 0 {
		return nil, fmt.Errorf("no hay datanodes configurados en %s", path)
	}

	addrs := make([]string, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		addr := fmt.Sprintf("%s:%d", n.Host, n.Port)
		addrs = append(addrs, addr)
	}

	return addrs, nil
}

type putPlanRequest struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
}

type putPlanResponse struct {
	Filename  string                   `json:"filename"`
	BlockSize int64                    `json:"block_size"`
	Blocks    []metadata.BlockLocation `json:"blocks"`
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("no se pudo cargar config.json: %v", err)
	}

	loggerAddr := cfg.Namenode.LoggerAddr
	if loggerAddr == "" {
		loggerAddr = ":5000"
	}

	if err := dfslog.Init("namenode", loggerAddr); err != nil {
		log.Printf("dfslog: no se pudo conectar al log server: %v", err)
	}
	dfslog.Infof("NameNode iniciando...")

	store := metadata.NewJSONStore("metadata.json")
	if err := store.Load(); err != nil {
		dfslog.Errorf("error cargando metadata: %v", err)
		log.Fatalf("error cargando metadata: %v", err)
	}

	dataNodes, err := loadDataNodesFromFile("datanodes.json")
	if err != nil {
		dfslog.Errorf("error cargando datanodes: %v", err)
		log.Fatal("error cargando datanodes:", err)
	}
	dfslog.Infof("DataNodes configurados: %v", dataNodes)

	nn := namenode.NewService(store, dataNodes)

	tcpAddr := cfg.Namenode.TCPAddr
	if tcpAddr == "" {
		tcpAddr = ":3000"
	}
	tcpServer := namenode.NewTCPServer(tcpAddr, nn)
	if err := tcpServer.Start(); err != nil {
		dfslog.Errorf("error iniciando servidor TCP del namenode: %v", err)
		log.Fatalf("error iniciando servidor TCP del namenode: %v", err)
	}
	dfslog.Infof("NameNode TCP escuchando en %s", tcpAddr)

	httpAddr := cfg.Namenode.HTTPAddr
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	http.HandleFunc("/files/plan", func(w http.ResponseWriter, r *http.Request) {
		dfslog.Infof("HTTP %s %s desde %s", r.Method, r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodPost {
			dfslog.Errorf("método no permitido en /files/plan: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req putPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			dfslog.Errorf("JSON inválido en /files/plan: %v", err)
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Filename == "" || req.SizeBytes <= 0 {
			dfslog.Errorf("request inválido en /files/plan: filename=%q size_bytes=%d", req.Filename, req.SizeBytes)
			http.Error(w, "filename y size_bytes son obligatorios", http.StatusBadRequest)
			return
		}

		blocks, err := nn.HandlePut(req.Filename, req.SizeBytes)
		if err != nil {
			dfslog.Errorf("error en HandlePut para %s: %v", req.Filename, err)
			http.Error(w, "error en HandlePut: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := putPlanResponse{
			Filename:  req.Filename,
			BlockSize: namenode.BlockSize,
			Blocks:    blocks,
		}

		dfslog.Infof("Plan PUT HTTP generado: filename=%s size_bytes=%d blocks=%d",
			req.Filename, req.SizeBytes, len(blocks))

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			dfslog.Errorf("error codificando respuesta en /files/plan: %v", err)
			log.Println("error codificando respuesta:", err)
		}
	})

	go func() {
		dfslog.Infof("NameNode HTTP escuchando en %s", httpAddr)
		if err := http.ListenAndServe(httpAddr, nil); err != nil {
			dfslog.Errorf("error en servidor HTTP: %v", err)
			log.Fatalf("error en servidor HTTP: %v", err)
		}
	}()

	select {}
}
