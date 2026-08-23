//go:build !cgo

package snapshot

import (
	"context"
	"database/sql"
	"errors"
)

// runBackupAPI is unreachable in practice: without cgo the sqlite3 driver
// never registers, so opening a connection fails before this runs. It
// exists so CGO_ENABLED=0 builds of the package stay compilable.
func runBackupAPI(context.Context, *sql.Conn, *sql.Conn) error {
	return errors.New("SQLite snapshots require a CGO build")
}
