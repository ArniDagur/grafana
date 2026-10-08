package sqlstore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestIsReadOnlyError(t *testing.T) {
	readOnly := &mysql.MySQLError{
		Number:  mysqlerr.ER_OPTION_PREVENTS_STATEMENT,
		Message: "The MySQL server is running with the --read-only option so it cannot execute this statement",
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "read_only", err: readOnly, want: true},
		{
			name: "super_read_only",
			err: &mysql.MySQLError{
				Number:  mysqlerr.ER_OPTION_PREVENTS_STATEMENT,
				Message: "The MySQL server is running with the --super-read-only option so it cannot execute this statement",
			},
			want: true,
		},
		{name: "read-only transaction", err: &mysql.MySQLError{Number: mysqlerr.ER_CANT_EXECUTE_IN_READ_ONLY_TRANSACTION}, want: true},
		{name: "read-only mode", err: &mysql.MySQLError{Number: mysqlerr.ER_READ_ONLY_MODE}, want: true},
		{name: "innodb_read_only", err: &mysql.MySQLError{Number: mysqlerr.ER_INNODB_READ_ONLY}, want: true},
		{name: "wrapped read-only error", err: fmt.Errorf("update token: %w", readOnly), want: true},
		{
			name: "other server option",
			err: &mysql.MySQLError{
				Number:  mysqlerr.ER_OPTION_PREVENTS_STATEMENT,
				Message: "The MySQL server is running with the --secure-file-priv option so it cannot execute this statement",
			},
			want: false,
		},
		{name: "duplicate entry", err: &mysql.MySQLError{Number: mysqlerr.ER_DUP_ENTRY}, want: false},
		{name: "non-MySQL error", err: errors.New("read-only"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsReadOnlyError(tt.err))
		})
	}
}
