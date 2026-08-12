package migrations

import (
	"strings"
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

func TestEmbeddedMigrationsIncludeOrganizationFoundation(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 53 {
			if migration.Name != "create_organizations" || !strings.Contains(migration.SQL, "ADD COLUMN organization_id") {
				t.Fatalf("unexpected organization migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("organization foundation migration not found")
}

func TestEmbeddedMigrationsIncludeCatalogStructure(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 54 {
			if migration.Name != "create_catalog_structure" || !strings.Contains(migration.SQL, "CREATE TABLE modalities") || !strings.Contains(migration.SQL, "CREATE TABLE spaces") {
				t.Fatalf("unexpected catalog migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("catalog structure migration not found")
}
