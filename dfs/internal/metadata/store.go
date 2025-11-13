package metadata

type Store interface {
	Load() error

	Save() error

	//consulta
	ListFiles() ([]string, error)
	GetFile(name string) ([]BlockLocation, bool)

	//modificación
	PutFile(name string, locations []BlockLocation) error
	DeleteFile(name string) error
}
