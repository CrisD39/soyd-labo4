package metadata

import (
	"encoding/json"
	"os"
	"sync"
)

type JSONStore struct {
	path string
	mtx  sync.Mutex // lo dejo mutex pero podria usar RWMutex
	data MetadataTable
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{
		path: path,
		data: make(MetadataTable),
	}
}

func (s *JSONStore) Load() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// El archivo no existe, inicializar tabla vacía
			s.data = make(MetadataTable)
			return nil
		}
		return err
	}

	// si existe pero está vacío, inicalizo tabla vacia
	if info.Size() == 0 {
		s.data = make(MetadataTable)
		return nil
	}

	content, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	// deserializar JSON
	s.data = make(MetadataTable)
	if err := json.Unmarshal(content, &s.data); err != nil { // deserializo en la estructura
		return err
	}

	return nil
}

func (s *JSONStore) Save() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	content, err := json.MarshalIndent(s.data, "", "  ") //serializo a JSON
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, content, 0644)
}

func (s *JSONStore) ListFiles() ([]string, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	files := make([]string, 0, len(s.data))
	for name := range s.data {
		files = append(files, name)
	}
	return files, nil
}

func (s *JSONStore) GetFile(name string) ([]BlockLocation, bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	locations, exists := s.data[name]
	return locations, exists
}

func (s *JSONStore) PutFile(name string, locations []BlockLocation) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.data[name] = locations
	return nil
}

func (s *JSONStore) DeleteFile(name string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.data, name)
	// idem: acá podrías llamar a s.Save()
	return nil
}
