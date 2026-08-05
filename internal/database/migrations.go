package database

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.up.sql
var migrationFS embed.FS

// migration represents a single schema migration step.
type migration struct {
	name string
	sql  string
}

// loadMigrations reads all embedded .up.sql files and returns them sorted by filename.
func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	migrations := make([]migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		data, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", e.Name(), err)
		}
		// Derive a human-readable name from the filename: strip prefix and suffix.
		name := strings.TrimSuffix(e.Name(), ".up.sql")
		// Strip the leading "000001_" prefix.
		if idx := strings.Index(name, "_"); idx != -1 {
			name = name[idx+1:]
		}
		migrations = append(migrations, migration{
			name: name,
			sql:  string(data),
		})
	}

	return migrations, nil
}
