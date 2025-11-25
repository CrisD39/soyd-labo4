package datanode

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"dfs/internal/logger" 
)

var ErrBlockNotFound = errors.New("block not found")

type DataNode interface {
	StoreBlock(blockID string, data []byte) error
	RetrieveBlock(blockID string) ([]byte, error)
}

type memoryDataNode struct {
	mtx    sync.RWMutex
	blocks map[string][]byte
}

func NewMemoryDataNode() DataNode {
	dfslog.Infof("Creando memoryDataNode (almacenamiento en memoria)")
	return &memoryDataNode{
		blocks: make(map[string][]byte),
	}
}


func (m *memoryDataNode) StoreBlock(blockID string, data []byte) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	buf := make([]byte, len(data))
	copy(buf, data)

	m.blocks[blockID] = buf
	dfslog.Debugf("memoryDataNode: StoreBlock id=%s size=%d", blockID, len(data))
	return nil
}


func (m *memoryDataNode) RetrieveBlock(blockID string) ([]byte, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	data, ok := m.blocks[blockID]
	if !ok {
		dfslog.Infof("memoryDataNode: bloque no encontrado id=%s", blockID)
		return nil, ErrBlockNotFound
	}

	buf := make([]byte, len(data))
	copy(buf, data)

	dfslog.Debugf("memoryDataNode: RetrieveBlock id=%s size=%d", blockID, len(buf))
	return buf, nil
}

type fileDataNode struct {
	baseDir string
	mtx     sync.RWMutex
}


func NewFileDataNode(baseDir string) (DataNode, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		dfslog.Errorf("fileDataNode: error creando baseDir=%s: %v", baseDir, err)
		return nil, err
	}
	dfslog.Infof("Creando fileDataNode con baseDir=%s", baseDir)
	return &fileDataNode{baseDir: baseDir}, nil
}

func (f *fileDataNode) blockPath(blockID string) string {
	return filepath.Join(f.baseDir, blockID)
}

func (f *fileDataNode) StoreBlock(blockID string, data []byte) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	path := f.blockPath(blockID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		dfslog.Errorf("fileDataNode: error guardando bloque id=%s path=%s: %v", blockID, path, err)
		return err
	}

	dfslog.Infof("fileDataNode: StoreBlock OK id=%s path=%s size=%d", blockID, path, len(data))
	return nil
}

func (f *fileDataNode) RetrieveBlock(blockID string) ([]byte, error) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()

	path := f.blockPath(blockID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			dfslog.Infof("fileDataNode: bloque no encontrado id=%s path=%s", blockID, path)
			return nil, ErrBlockNotFound
		}
		dfslog.Errorf("fileDataNode: error leyendo bloque id=%s path=%s: %v", blockID, path, err)
		return nil, err
	}

	dfslog.Debugf("fileDataNode: RetrieveBlock OK id=%s path=%s size=%d", blockID, path, len(data))
	return data, nil
}
