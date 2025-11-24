package namenode

import (
	"fmt"

	// 👈 nuevo import
	dfslog "dfs/internal/logger"
	"dfs/internal/metadata"
)

// Tamaño de bloque que usará el namenode para planificar (1 KB por ahora).
// El cliente puede leer este valor vía la API HTTP.
const (
	BlockSize         int64 = 1024
	ReplicationFactor       = 3 // cuántas réplicas por bloque queremos
)

// Namenode expone las operaciones de alto nivel. La capa de red (HTTP/TCP)
// llama a estos métodos y luego serializa la respuesta a JSON/texto.
type Namenode interface {
	// HandlePut planifica dónde irán los bloques de un archivo nuevo.
	// Devuelve, para cada bloque, su ID y la lista de DataNodes donde debe guardarse.
	HandlePut(filename string, sizeBytes int64) ([]metadata.BlockLocation, error)

	// HandleGet devuelve el plan de bloques previamente guardado para un archivo.
	// El bool indica si el archivo existe en el store.
	HandleGet(filename string) ([]metadata.BlockLocation, bool, error)

	// HandleInfo por ahora es igual a Get, pero podría devolver más metadatos.
	HandleInfo(filename string) ([]metadata.BlockLocation, bool, error)

	// HandleList devuelve la lista de archivos conocidos por el NameNode.
	HandleList() ([]string, error)
}

// Cada metadata.BlockLocation debería verse (en el paquete metadata) algo así:
//
//  type BlockLocation struct {
//      BlockID  string   `json:"block_id"`
//      Replicas []string `json:"replicas"` // lista de DataNodes tipo "host:puerto"
//  }
//
// Es decir, ya no tenemos un solo Node, sino varias réplicas.

type namenode struct {
	store     metadata.Store
	dataNodes []string // lista de datanodes como "host:puerto"
	nextIndex int      // índice para hacer round-robin
}

// NewService crea un namenode con un store de metadatos y la lista de DataNodes
// disponibles.
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

// pickDataNodes devuelve hasta n DataNodes distintos usando round-robin.
// Si hay menos DataNodes que n, devuelve todos sin repetir.
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

	// Avanzamos el índice global para repartir los próximos bloques.
	s.nextIndex = (s.nextIndex + n) % len(s.dataNodes)

	return replicas
}

// HandlePut calcula cuántos bloques tendrá el archivo y, para cada uno,
// elige ReplicationFactor DataNodes donde guardarlo.
func (s *namenode) HandlePut(filename string, sizeBytes int64) ([]metadata.BlockLocation, error) {
	if sizeBytes <= 0 {
		dfslog.Errorf("HandlePut: tamaño inválido para %s (size=%d)", filename, sizeBytes)
		return nil, fmt.Errorf("file size must be positive")
	}

	dfslog.Infof("HandlePut: filename=%s size=%d", filename, sizeBytes)

	// Cantidad de bloques, redondeando hacia arriba:
	//   numBlocks = ceil(sizeBytes / BlockSize)
	numBlocks64 := (sizeBytes + BlockSize - 1) / BlockSize
	numBlocks := int(numBlocks64)

	locations := make([]metadata.BlockLocation, 0, numBlocks)

	for i := 0; i < numBlocks; i++ {
		blockID := fmt.Sprintf("%s_block_%d", filename, i+1)

		// Elegimos los DataNodes que almacenarán este bloque.
		replicas := s.pickDataNodes(ReplicationFactor) // ej. ["127.0.0.1:4001", "127.0.0.1:4002", ...]

		loc := metadata.BlockLocation{
			BlockID:  blockID,
			Replicas: replicas,
		}
		locations = append(locations, loc)
	}

	dfslog.Infof("HandlePut: archivo %s se partió en %d bloques", filename, len(locations))

	// Guardamos en el store el plan ya decidido para este archivo.
	if err := s.store.PutFile(filename, locations); err != nil {
		dfslog.Errorf("HandlePut: error guardando metadata para %s: %v", filename, err)
		return nil, err
	}
	// Save no devuelve error en tu implementación actual, pero igual lo dejamos logueado.
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
