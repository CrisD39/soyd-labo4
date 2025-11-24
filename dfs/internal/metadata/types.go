package metadata

//Localización de un bloque de archivo en el sistema distribuido
// internal/metadata/types.go (por ejemplo)
type BlockLocation struct {
	BlockID  string   `json:"block_id"`
	Replicas []string `json:"replicas"` // ahora es una lista de DataNodes
}

type MetadataTable map[string][]BlockLocation //Mapa que asocia un ID de archivo con una lista de ubicaciones de bloques
