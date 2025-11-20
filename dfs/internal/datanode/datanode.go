package datanode

type DataNode interface {
	StoreBlock(blockID string, data []byte) error
	RetrieveBlock(blockID string) ([]byte, error)
}