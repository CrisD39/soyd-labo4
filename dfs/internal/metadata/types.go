package metadata

//Localización de un bloque de archivo en el sistema distribuido
type BlockLocation struct {
	BlockID string `json:"block_id"`
	Node    string `json:"node"`
}

type MetadataTable map[string][]BlockLocation //Mapa que asocia un ID de archivo con una lista de ubicaciones de bloques
