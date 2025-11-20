package namenode

import (
	"fmt"

	"dfs/internal/metadata"
)

type Namenode interface {
	HandlePut(filename string, sizeBytes int64) ([]metadata.BlockLocation, error)
	HandleGet(filename string) ([]metadata.BlockLocation, bool, error)
	HandleInfo(filename string) ([]metadata.BlockLocation, bool, error)
	HandleList() ([]string, error)
}

// implementación concreta de Service.
type namenode struct {
	store metadata.Store
}

func NewService(store metadata.Store) Namenode {
	return &namenode{store: store}
}

func (s *namenode) HandlePut(filename string, sizeBytes int64) ([]metadata.BlockLocation, error) {
	// tamaño de bloque
	const blockSize int64 = 1024 // 1 KB

	if sizeBytes <= 0 {
		return nil, fmt.Errorf("file size must be positive")
	}

	// cantidad de bloques, redondeando hacia arriba
	numBlocks64 := (sizeBytes + blockSize - 1) / blockSize
	numBlocks := int(numBlocks64)

	locations := make([]metadata.BlockLocation, 0, numBlocks)

	for i := 0; i < numBlocks; i++ {
		blockID := fmt.Sprintf("%s_block_%d", filename, i+1)

		location := metadata.BlockLocation{
			BlockID: blockID,
			// TODO: elegir un datanode real
			Node: "node_placeholder",
		}
		locations = append(locations, location)
	}

	// guardo en el store
	if err := s.store.PutFile(filename, locations); err != nil {
		return nil, err
	}
	s.store.Save()
	return locations, nil
}

func (s *namenode) HandleGet(filename string) ([]metadata.BlockLocation, bool, error) {
	locations, exists := s.store.GetFile(filename)
	return locations, exists, nil
}

func (s *namenode) HandleInfo(filename string) ([]metadata.BlockLocation, bool, error) {
	// va a ser lo mismo que el get por ahora, leugo con el protocolo cambiaria creo
	locations, exists := s.store.GetFile(filename)
	return locations, exists, nil
}

func (s *namenode) HandleList() ([]string, error) {
	return s.store.ListFiles()
}
