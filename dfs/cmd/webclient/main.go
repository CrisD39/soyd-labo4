package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"dfs/internal/client"
	dfslog "dfs/internal/logger"
)

func main() {
	if err := dfslog.Init("webclient", "localhost:7000"); err != nil {
		log.Printf("dfslog: no se pudo conectar al log server: %v", err)
	}
	dfslog.Infof("webclient iniciado")

	c := client.NewTCPClient("localhost:3000")

	http.HandleFunc("/api/ls", func(w http.ResponseWriter, r *http.Request) {
		dfslog.Infof("HTTP %s %s desde %s", r.Method, r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := c.List()
		if err != nil {
			dfslog.Errorf("error en c.List(): %v", err)
			http.Error(w, "error en ls: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := struct {
			Files []string `json:"files"`
		}{Files: files}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			dfslog.Errorf("error encodeando JSON en /api/ls: %v", err)
		}
	})

	http.HandleFunc("/files/download", downloadHandler(c))

	http.HandleFunc("/files/upload", uploadHandler(c))

	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	addr := ":8081"
	log.Printf("webclient escuchando en http://localhost%s", addr)
	dfslog.Infof("webclient HTTP escuchando en %s", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func uploadHandler(c client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dfslog.Infof("HTTP %s %s desde %s", r.Method, r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			dfslog.Errorf("ParseMultipartForm error: %v", err)
			http.Error(w, "invalid multipart/form-data: "+err.Error(), http.StatusBadRequest)
			return
		}

		remoteName := r.FormValue("remote")
		if remoteName == "" {
			http.Error(w, "missing 'remote' field", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			dfslog.Errorf("FormFile error: %v", err)
			http.Error(w, "missing file field: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		dfslog.Infof("Subiendo archivo %s como %s (size reportado: %d)",
			header.Filename, remoteName, header.Size)

		tmpFile, err := os.CreateTemp("", "dfs-upload-*")
		if err != nil {
			dfslog.Errorf("CreateTemp error: %v", err)
			http.Error(w, "cannot create temp file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpPath := tmpFile.Name()

		if _, err := io.Copy(tmpFile, file); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			dfslog.Errorf("error copiando upload a temp: %v", err)
			http.Error(w, "cannot save uploaded file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpFile.Close()

		if err := c.Put(tmpPath, remoteName); err != nil {
			os.Remove(tmpPath)
			dfslog.Errorf("error en c.Put(%s, %s): %v", tmpPath, remoteName, err)
			http.Error(w, "error subiendo al DFS: "+err.Error(), http.StatusBadGateway)
			return
		}

		if err := os.Remove(tmpPath); err != nil {
			dfslog.Infof("WARN: no se pudo borrar temp %s: %v", tmpPath, err)
		}

		dfslog.Infof("PUT completado (web): archivo %s subido como %s", header.Filename, remoteName)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"remote": remoteName,
		}); err != nil {
			dfslog.Errorf("error encodeando JSON en /files/upload: %v", err)
		}
	}
}

func downloadHandler(c client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dfslog.Infof("HTTP %s %s desde %s", r.Method, r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		remote := r.URL.Query().Get("name")
		if remote == "" {
			http.Error(w, "falta parámetro 'name'", http.StatusBadRequest)
			return
		}

		tmpFile, err := os.CreateTemp("", "dfs-download-*")
		if err != nil {
			http.Error(w, "no se pudo crear archivo temporal", http.StatusInternalServerError)
			return
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath)

		if err := c.Get(remote, tmpPath); err != nil {
			dfslog.Errorf("error en c.Get(%s, %s): %v", remote, tmpPath, err)
			http.Error(w, "error descargando desde DFS: "+err.Error(), http.StatusBadGateway)
			return
		}

		f, err := os.Open(tmpPath)
		if err != nil {
			http.Error(w, "no se pudo abrir archivo temporal", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		size := stat.Size()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=%q", filepath.Base(remote)))
		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}

		if _, err := io.Copy(w, f); err != nil {
			dfslog.Errorf("error enviando archivo %s al navegador: %v", remote, err)
			return
		}
	}
}
