package migrations

import (
	"testing"
	"testing/fstest"
)

func TestLoadOrdersAndChecksumsMigrations(t *testing.T) {
	files := fstest.MapFS{
		"002_second.sql": {Data: []byte("SELECT 2;")},
		"001_first.sql":  {Data: []byte("SELECT 1;")},
	}
	loaded, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].Version != 1 || loaded[1].Version != 2 {
		t.Fatalf("unexpected migration order: %+v", loaded)
	}
	if loaded[0].Checksum == "" || loaded[0].Checksum == loaded[1].Checksum {
		t.Fatal("expected stable distinct checksums")
	}
}

func TestLoadRejectsSequenceGap(t *testing.T) {
	_, err := Load(fstest.MapFS{"002_second.sql": {Data: []byte("SELECT 2;")}})
	if err == nil {
		t.Fatal("expected a sequence gap error")
	}
}

func TestLoadRejectsInvalidFilename(t *testing.T) {
	_, err := Load(fstest.MapFS{"migration.sql": {Data: []byte("SELECT 1;")}})
	if err == nil {
		t.Fatal("expected an invalid filename error")
	}
}
