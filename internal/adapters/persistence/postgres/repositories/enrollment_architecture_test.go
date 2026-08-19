package repositories

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnrollmentRepositoryDoesNotQueryPeopleTables(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}
	payload, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "enrollment_repository.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ToLower(string(payload))
	for _, table := range []string{"students", "persons", "person_unit_relationships"} {
		if strings.Contains(source, table) {
			t.Fatalf("enrollment repository must use the people port instead of querying %s", table)
		}
	}
}
