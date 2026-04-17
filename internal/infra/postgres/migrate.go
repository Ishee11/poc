package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsFS fs.FS) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}

		query, err := fs.ReadFile(migrationsFS, "migrations/"+e.Name())
		if err != nil {
			return err
		}

		if _, err := pool.Exec(ctx, string(query)); err != nil {
			return fmt.Errorf("migration %s failed: %w", e.Name(), err)
		}
	}

	return nil
}
