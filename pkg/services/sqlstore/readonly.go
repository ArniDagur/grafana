package sqlstore

import (
	"errors"
	"strings"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
)

// IsReadOnlyError reports whether err means the database rejected a write because it is read-only,
// for example a MySQL primary that was switched to read_only or super_read_only during a failover.
func IsReadOnlyError(err error) bool {
	mysqlErr, ok := errors.AsType[*mysql.MySQLError](err)
	if !ok {
		return false
	}

	switch mysqlErr.Number {
	case mysqlerr.ER_OPTION_PREVENTS_STATEMENT:
		// The same code is used for other server options such as --secure-file-priv.
		return strings.Contains(mysqlErr.Message, "read-only")
	case mysqlerr.ER_CANT_EXECUTE_IN_READ_ONLY_TRANSACTION,
		mysqlerr.ER_READ_ONLY_MODE,
		mysqlerr.ER_INNODB_READ_ONLY:
		return true
	default:
		return false
	}
}
