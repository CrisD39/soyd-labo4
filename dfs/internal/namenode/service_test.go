package namenode

import (
	"dfs/internal/metadata"
	"reflect"
	"testing"
)

// ----------------- helpers -----------------

// fakeStore implementa metadata.Store, pero todo en memoria
type fakeStore struct {
	data metadata.MetadataTable
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		data: make(metadata.MetadataTable),
	}
}

func (f *fakeStore) Load() error { return nil }
func (f *fakeStore) Save() error { return nil }
func (f *fakeStore) ListFiles() ([]string, error) {
	files := make([]string, 0, len(f.data))
	for name := range f.data {
		files = append(files, name)
	}
	return files, nil
}
func (f *fakeStore) GetFile(name string) ([]metadata.BlockLocation, bool) {
	locs, ok := f.data[name]
	return locs, ok
}
func (f *fakeStore) PutFile(name string, blocks []metadata.BlockLocation) error {
	f.data[name] = blocks
	return nil
}
func (f *fakeStore) DeleteFile(name string) error {
	delete(f.data, name)
	return nil
}

// assert genérico tipo assert.Equals
func assertEqual[T comparable](t *testing.T, want, got T, msg string) {
	t.Helper()
	if want != got {
		t.Fatalf("%s: want=%v got=%v", msg, want, got)
	}
}

func assertDeepEqual(t *testing.T, want, got any, msg string) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s: want=%v got=%v", msg, want, got)
	}
}

// ----------------- tests -----------------

// 1) HandlePut: calcula bien la cantidad de bloques y guarda en el store
func TestHandlePutBlocksAndStore(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	filename := "file.txt"

	// 1 KB → 1 bloque (blockSize = 1024)
	locs, err := svc.HandlePut(filename, 1024)
	if err != nil {
		t.Fatalf("HandlePut returned error: %v", err)
	}
	assertEqual(t, 1, len(locs), "expected 1 block for 1024 bytes")

	assertEqual(t, "file.txt_block_1", locs[0].BlockID, "block id mismatch")
	assertEqual(t, "node_placeholder", locs[0].Node, "node placeholder mismatch")

	// Verificamos que se haya guardado en el store
	stored, ok := store.GetFile(filename)
	if !ok {
		t.Fatalf("file %s not found in store after HandlePut", filename)
	}
	assertDeepEqual(t, locs, stored, "stored locations should match returned locations")

	// 1025 bytes → 2 bloques
	locs2, err := svc.HandlePut("big.txt", 1025)
	if err != nil {
		t.Fatalf("HandlePut (1025 bytes) returned error: %v", err)
	}
	assertEqual(t, 2, len(locs2), "expected 2 blocks for 1025 bytes")
}

// 2) HandlePut con tamaño inválido
func TestHandlePutInvalidSize(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.HandlePut("bad.txt", 0); err == nil {
		t.Fatalf("expected error for non-positive size, got nil")
	}
}

// 3) HandleGet: debe devolver lo que está en el store
func TestHandleGet(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	filename := "example.txt"
	expected := []metadata.BlockLocation{
		{BlockID: "example.txt_block_1", Node: "node1"},
		{BlockID: "example.txt_block_2", Node: "node2"},
	}

	// precargamos el store a mano
	if err := store.PutFile(filename, expected); err != nil {
		t.Fatalf("PutFile on fakeStore returned error: %v", err)
	}

	locs, exists, err := svc.HandleGet(filename)
	if err != nil {
		t.Fatalf("HandleGet returned error: %v", err)
	}
	if !exists {
		t.Fatalf("HandleGet says file does not exist, but it was preloaded")
	}
	assertDeepEqual(t, expected, locs, "HandleGet locations mismatch")

	// archivo inexistente
	_, exists, err = svc.HandleGet("nope.txt")
	if err != nil {
		t.Fatalf("HandleGet on non-existing file returned error: %v", err)
	}
	if exists {
		t.Fatalf("HandleGet should report exists=false for non-existing file")
	}
}

// 4) HandleInfo: mismo comportamiento básico que HandleGet
func TestHandleInfo(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	filename := "info.txt"
	expected := []metadata.BlockLocation{
		{BlockID: "info.txt_block_1", Node: "nodeX"},
	}

	if err := store.PutFile(filename, expected); err != nil {
		t.Fatalf("PutFile on fakeStore returned error: %v", err)
	}

	locs, exists, err := svc.HandleInfo(filename)
	if err != nil {
		t.Fatalf("HandleInfo returned error: %v", err)
	}
	if !exists {
		t.Fatalf("HandleInfo says file does not exist, but it was preloaded")
	}
	assertDeepEqual(t, expected, locs, "HandleInfo locations mismatch")
}

// 5) HandleList: devuelve la lista de archivos que ve el store
func TestHandleList(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_ = store.PutFile("a.txt", nil)
	_ = store.PutFile("b.txt", nil)

	files, err := svc.HandleList()
	if err != nil {
		t.Fatalf("HandleList returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("HandleList expected 2 files, got %d (%v)", len(files), files)
	}

	// No asumimos orden, usamos un map
	seen := make(map[string]bool)
	for _, f := range files {
		seen[f] = true
	}

	if !seen["a.txt"] || !seen["b.txt"] {
		t.Fatalf("HandleList result missing expected files: %v", files)
	}
}
