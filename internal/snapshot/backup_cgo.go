//go:build cgo

package snapshot

import (
	"context"
	"database/sql"
	"fmt"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// runBackupAPI streams srcConn's main database into dstConn through the
// SQLite online backup API, honoring ctx between page batches. Both
// connections must already be dedicated (see backupInto).
func runBackupAPI(ctx context.Context, srcConn, dstConn *sql.Conn) error {
	return dstConn.Raw(func(driverDst any) error {
		dstSQLite, ok := driverDst.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected destination driver conn %T", driverDst)
		}
		var srcSQLite *sqlite3.SQLiteConn
		if err := srcConn.Raw(func(driverSrc any) error {
			srcSQLite, ok = driverSrc.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("unexpected source driver conn %T", driverSrc)
			}
			return nil
		}); err != nil {
			return err
		}

		backup, err := dstSQLite.Backup("main", srcSQLite, "main")
		if err != nil {
			return err
		}
		for {
			select {
			case <-ctx.Done():
				_ = backup.Finish()
				return ctx.Err()
			default:
			}
			done, stepErr := backup.Step(256)
			if stepErr != nil {
				_ = backup.Finish()
				return stepErr
			}
			if done {
				return backup.Finish()
			}
		}
	})
}
