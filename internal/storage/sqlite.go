package storage

import (
	"fmt"
)

// SQLiteDeprecated: Use PostgreSQL instead
// This file is kept for reference only
// NewSQLite is deprecated and should not be used
// Deprecated: Use NewPostgres instead
func NewSQLite(dbPath string) (interface{}, error) {
	return nil, fmt.Errorf("SQLite support has been deprecated. Use PostgreSQL instead")
}
