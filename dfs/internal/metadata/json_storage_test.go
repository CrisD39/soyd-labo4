package metadata

import (
	"path/filepath"
	"reflect"
	"testing"
)

// Helper para crear un JSONStore en un directorio temporal
func newTestStore(t *testing.T) (*JSONStore, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.json")
	store := NewJSONStore(path)
	return store, path
}

// Helper tipo assert.Equals
func assertEqual[T comparable](t *testing.T, want, got T, msg string) {
	t.Helper()
	if want != got {
		t.Fatalf("%s: want %v, got %v", msg, want, got)
	}
}

// ---------- TESTS ----------

// 1) Load sobre un archivo que NO existe
func TestLoadNonExistingFile(t *testing.T) {
	store, _ := newTestStore(t)

	// Todavía no hay archivo en disco
	if err := store.Load(); err != nil {
		t.Fatalf("Load should not fail for non-existing file: %v", err)
	}

	files, err := store.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles after Load (no file) returned error: %v", err)
	}

	assertEqual(t, 0, len(files), "expected empty metadata after Load with no file")
}

// 2) PutFile + GetFile + ListFiles en memoria
func TestPutGetListDeleteFile(t *testing.T) {
	store, _ := newTestStore(t)

	filename := "example.txt"
	locs := []BlockLocation{
		{BlockID: "b1", Node: "node1"},
		{BlockID: "b2", Node: "node2"},
	}

	// PutFile
	if err := store.PutFile(filename, locs); err != nil {
		t.Fatalf("PutFile returned error: %v", err)
	}

	// ListFiles
	files, err := store.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	assertEqual(t, 1, len(files), "expected 1 file in metadata")
	assertEqual(t, filename, files[0], "expected file name to match")

	// GetFile
	gotLocs, ok := store.GetFile(filename)
	if !ok {
		t.Fatalf("GetFile should find %s but did not", filename)
	}
	if !reflect.DeepEqual(locs, gotLocs) {
		t.Fatalf("GetFile returned wrong locations. want=%v got=%v", locs, gotLocs)
	}

	// DeleteFile
	if err := store.DeleteFile(filename); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}

	files, err = store.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles after delete returned error: %v", err)
	}
	assertEqual(t, 0, len(files), "expected 0 files after delete")
}

// 3) Save + Load: comprobar que persiste en disco correctamente
func TestSaveAndLoadRoundTrip(t *testing.T) {
	store, path := newTestStore(t)

	filename := "example.txt"
	locs := []BlockLocation{
		{BlockID: "b1", Node: "node1"},
		{BlockID: "b2", Node: "node2"},
	}

	// Guardamos metadata en el store
	if err := store.PutFile(filename, locs); err != nil {
		t.Fatalf("PutFile returned error: %v", err)
	}

	// Persistimos en disco
	if err := store.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Creamos un store nuevo que apunta al MISMO archivo
	store2 := NewJSONStore(path)

	// Cargamos desde disco
	if err := store2.Load(); err != nil {
		t.Fatalf("Load on second store returned error: %v", err)
	}

	// Verificamos que tenga el mismo contenido
	gotLocs, ok := store2.GetFile(filename)
	if !ok {
		t.Fatalf("GetFile on second store should find %s but did not", filename)
	}
	if !reflect.DeepEqual(locs, gotLocs) {
		t.Fatalf("second store has wrong locations. want=%v got=%v", locs, gotLocs)
	}
}
