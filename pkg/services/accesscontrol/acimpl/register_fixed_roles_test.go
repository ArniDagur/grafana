package acimpl

import (
	"context"
	"errors"
	"testing"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/accesscontrol/permreg"
	"github.com/grafana/grafana/pkg/services/accesscontrol/seeding"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/setting"
)

// failingSeedStore returns no stored roles and rejects every write with writeErr.
type failingSeedStore struct {
	writeErr error
	writes   int
}

func (s *failingSeedStore) write() error {
	s.writes++
	return s.writeErr
}

func (s *failingSeedStore) LoadRoles(context.Context) (map[string]*accesscontrol.RoleDTO, error) {
	return map[string]*accesscontrol.RoleDTO{}, nil
}

func (s *failingSeedStore) SetRole(context.Context, *accesscontrol.RoleDTO, accesscontrol.RoleDTO) error {
	return s.write()
}

func (s *failingSeedStore) SetPermissions(context.Context, *accesscontrol.RoleDTO, accesscontrol.RoleDTO) error {
	return s.write()
}

func (s *failingSeedStore) CreateRole(context.Context, accesscontrol.RoleDTO) error {
	return s.write()
}

func (s *failingSeedStore) DeleteRoles(context.Context, []string) error {
	return s.write()
}

func (s *failingSeedStore) LoadPrevious(context.Context) (map[accesscontrol.SeedPermission]struct{}, error) {
	return map[accesscontrol.SeedPermission]struct{}{}, nil
}

func (s *failingSeedStore) Apply(context.Context, []accesscontrol.SeedPermission, []accesscontrol.SeedPermission, map[accesscontrol.SeedPermission]accesscontrol.SeedPermission) error {
	return s.write()
}

func TestService_RegisterFixedRoles_DatabaseWriteFailure(t *testing.T) {
	newService := func(t *testing.T, store *failingSeedStore) *Service {
		t.Helper()
		ac := &Service{
			cfg:           setting.NewCfg(),
			log:           log.New("accesscontrol"),
			registrations: accesscontrol.RegistrationList{},
			roles:         accesscontrol.BuildBasicRoleDefinitions(),
			permRegistry:  permreg.ProvidePermissionRegistry(),
			seeder:        seeding.New(log.New("accesscontrol.seeder"), store, store),
		}
		require.NoError(t, ac.DeclareFixedRoles(accesscontrol.RoleRegistration{
			Role: accesscontrol.RoleDTO{
				Name:        "fixed:test:role",
				Permissions: []accesscontrol.Permission{{Action: "test:read"}},
			},
			Grants: []string{string(org.RoleViewer)},
		}))
		return ac
	}

	t.Run("starts with in-memory roles when the database is read-only", func(t *testing.T) {
		store := &failingSeedStore{writeErr: &mysql.MySQLError{
			Number:  mysqlerr.ER_OPTION_PREVENTS_STATEMENT,
			Message: "The MySQL server is running with the --super-read-only option so it cannot execute this statement",
		}}
		ac := newService(t, store)

		require.NoError(t, ac.RegisterFixedRoles(context.Background()))

		assert.Equal(t, 1, store.writes, "seeding should have attempted a write")
		assert.True(t, ac.isInitialized)
		assert.Contains(t, ac.roles[string(org.RoleViewer)].Permissions, accesscontrol.Permission{Action: "test:read"})
	})

	t.Run("fails on other database errors", func(t *testing.T) {
		writeErr := errors.New("connection refused")
		ac := newService(t, &failingSeedStore{writeErr: writeErr})

		require.ErrorIs(t, ac.RegisterFixedRoles(context.Background()), writeErr)
	})
}
