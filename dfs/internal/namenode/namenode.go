package namenode

import (
	"fmt"

	dfslog "dfs/internal/logger"
	"dfs/internal/metadata"
)

const (
	BlockSize         int64 = 1024
	ReplicationFactor       = 3
)

type Namenode interface {
	HandlePut(filename string, sizeBytes int64) ([]metadata.BlockLocation, error)
	HandleGet(filename string) ([]metadata.BlockLocation, bool, error)
	HandleInfo(filename string) ([]metadata.BlockLocation, bool, error)
	HandleList() ([]string, error)
	HandleDelete(filename string) (bool, error)
}

type namenode struct {
	store     metadata.Store
	dataNodes []string
	nextIndex int
}

func NewService(store metadata.Store, dataNodes []string) Namenode {
	if len(dataNodes) == 0 {
		panic("namenode: se requiere al menos un datanode")
	}

	dfslog.Infof("Creando servicio NameNode con %d DataNodes", len(dataNodes))

	return &namenode{
		store:     store,
		dataNodes: dataNodes,
		nextIndex: 0,
	}
}

func (s *namenode) pickDataNodes(n int) []string {
	if len(s.dataNodes) == 0 || n <= 0 {
		return nil
	}

	if n > len(s.dataNodes) {
		n = len(s.dataNodes)
	}

	replicas := make([]string, 0, n)

	for i := 0; i < n; i++ {
		idx := (s.nextIndex + i) % len(s.dataNodes)
		replicas = append(replicas, s.dataNodes[idx])
	}

	s.nextIndex = (s.nextIndex + n) % len(s.dataNodes)

	return replicas
}

func (s *namenode) HandlePut(filename string, sizeBytes int64) ([]metadata.BlockLocation, error) {
	if sizeBytes <= 0 {
		dfslog.Errorf("HandlePut: tamaño inválido para %s (size=%d)", filename, sizeBytes)
		return nil, fmt.Errorf("file size must be positive")
	}

	dfslog.Infof("HandlePut: filename=%s size=%d", filename, sizeBytes)

	numBlocks64 := (sizeBytes + BlockSize - 1) / BlockSize
	numBlocks := int(numBlocks64)

	locations := make([]metadata.BlockLocation, 0, numBlocks)

	for i := 0; i < numBlocks; i++ {
		blockID := fmt.Sprintf("%s_block_%d", filename, i+1)

		replicas := s.pickDataNodes(ReplicationFactor)

		loc := metadata.BlockLocation{
			BlockID:  blockID,
			Replicas: replicas,
		}
		locations = append(locations, loc)
	}

	dfslog.Infof("HandlePut: archivo %s se partió en %d bloques", filename, len(locations))

	if err := s.store.PutFile(filename, locations); err != nil {
		dfslog.Errorf("HandlePut: error guardando metadata para %s: %v", filename, err)
		return nil, err
	}

	s.store.Save()
	dfslog.Infof("HandlePut: metadata de %s persistida correctamente", filename)

	return locations, nil
}

func (s *namenode) HandleGet(filename string) ([]metadata.BlockLocation, bool, error) {
	dfslog.Infof("HandleGet: filename=%s", filename)
	locations, exists := s.store.GetFile(filename)
	if !exists {
		dfslog.Infof("HandleGet: archivo %s no existe", filename)
	} else {
		dfslog.Debugf("HandleGet: archivo %s tiene %d bloques", filename, len(locations))
	}
	return locations, exists, nil
}

func (s *namenode) HandleInfo(filename string) ([]metadata.BlockLocation, bool, error) {
	dfslog.Infof("HandleInfo: filename=%s", filename)
	locations, exists := s.store.GetFile(filename)
	if !exists {
		dfslog.Infof("HandleInfo: archivo %s no existe", filename)
	} else {
		dfslog.Debugf("HandleInfo: archivo %s tiene %d bloques", filename, len(locations))
	}
	return locations, exists, nil
}

func (s *namenode) HandleList() ([]string, error) {
	dfslog.Infof("HandleList: listando archivos")
	files, err := s.store.ListFiles()
	if err != nil {
		dfslog.Errorf("HandleList: error listando archivos: %v", err)
		return nil, err
	}
	dfslog.Infof("HandleList: se encontraron %d archivos", len(files))
	return files, nil
}

func (s *namenode) HandleDelete(filename string) (bool, error) {
	dfslog.Infof("HandleDelete: filename=%s", filename)

	_, exists := s.store.GetFile(filename)
	if !exists {
		dfslog.Infof("HandleDelete: archivo %s no existe", filename)
		return false, nil
	}

	if err := s.store.DeleteFile(filename); err != nil {
		dfslog.Errorf("HandleDelete: error borrando metadata para %s: %v", filename, err)
		return false, err
	}

	if err := s.store.Save(); err != nil {
		dfslog.Errorf("HandleDelete: error persistiendo metadata para %s: %v", filename, err)
		return false, err
	}

	dfslog.Infof("HandleDelete: archivo %s eliminado de metadata", filename)
	return true, nil
}
