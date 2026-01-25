package infrastructure

import (
	"strings"
	"testing"
	"time"

	"business/internal/library/logger"
	"business/internal/library/mysql"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type authRepoTestEnv struct {
	repo   *Repository
	db     *gorm.DB
	clean  func() error
	nowUTC time.Time
}

func newAuthRepoTestEnv(t *testing.T) *authRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfAuthDBUnavailable(t, err)
	}
	require.NoError(t, err)

	err = mysqlConn.DB.AutoMigrate(&userRecord{}, &emailVerificationTokenRecord{})
	require.NoError(t, err)

	return &authRepoTestEnv{
		repo:   NewRepository(mysqlConn.DB, logger.NewNop()),
		db:     mysqlConn.DB,
		clean:  cleanup,
		nowUTC: time.Now().UTC().Truncate(time.Second),
	}
}

func (env *authRepoTestEnv) insertUser(t *testing.T, email string) {
	t.Helper()
	err := env.db.Create(&userRecord{
		Name:      "test-user",
		Email:     email,
		Password:  "hashed-password",
		CreatedAt: env.nowUTC,
		UpdatedAt: env.nowUTC,
	}).Error
	require.NoError(t, err)
}

func skipIfAuthDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping auth repository integration test: %v", err)
	}
}
