package app

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestMalformedEventIDsAreNotFoundBeforeDatabaseAccess(t *testing.T) {
	application := &App{}
	for _, id := range []string{"", "does-not-exist", "not-a-uuid", "00000000-0000-0000-0000-zzzzzzzzzzzz"} {
		t.Run(id, func(t *testing.T) {
			if _, err := application.loadEventDetails(t.Context(), id); !errors.Is(err, pgx.ErrNoRows) {
				t.Fatalf("load error = %v, want not found", err)
			}
			if _, err := application.loadEventDetailsForUpdate(t.Context(), nil, id); !errors.Is(err, pgx.ErrNoRows) {
				t.Fatalf("locking load error = %v, want not found", err)
			}
		})
	}
}
