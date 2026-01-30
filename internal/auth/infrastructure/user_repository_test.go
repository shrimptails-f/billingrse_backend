package infrastructure

import (
	"context"
	"testing"
	"time"

	"business/internal/auth/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepository_GetUserByEmail_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "user@example.com")

	got, err := env.repo.GetUserByEmail(context.Background(), domain.EmailAddress("user@example.com"))

	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", got.Email.String())
	assert.Equal(t, "hashed-password", got.PasswordHash.String())
	assert.WithinDuration(t, env.nowUTC, got.CreatedAt, time.Second)
	assert.WithinDuration(t, env.nowUTC, got.UpdatedAt, time.Second)
}

func TestRepository_GetUserByEmail_NotFound(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	_, err := env.repo.GetUserByEmail(context.Background(), domain.EmailAddress("missing@example.com"))

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_GetUserByEmail_NilContext(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "nilctx@example.com")
	ctx := context.Background()

	user, err := env.repo.GetUserByEmail(ctx, domain.EmailAddress("nilctx@example.com"))

	assert.NoError(t, err)
	assert.Equal(t, "nilctx@example.com", user.Email.String())
}

func TestRepository_CreateUser_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	user, err := env.repo.CreateUser(context.Background(), domain.User{
		Name:         domain.UserName("New User"),
		Email:        domain.EmailAddress("new@example.com"),
		PasswordHash: domain.NewPasswordHashFromHash("hashed"),
	})

	assert.NoError(t, err)
	assert.Equal(t, "New User", user.Name.String())
	assert.Equal(t, "new@example.com", user.Email.String())

	var record userRecord
	err = env.db.Where("email = ?", "new@example.com").First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, "New User", record.Name)
	assert.Equal(t, "hashed", record.Password)
}

func TestRepository_CreateUser_DuplicateEmail(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "dup@example.com")

	_, err := env.repo.CreateUser(context.Background(), domain.User{
		Name:         domain.UserName("Dup User"),
		Email:        domain.EmailAddress("dup@example.com"),
		PasswordHash: domain.NewPasswordHashFromHash("hashed"),
	})

	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestRepository_GetUserByID_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "getbyid@example.com")

	var record userRecord
	err := env.db.Where("email = ?", "getbyid@example.com").First(&record).Error
	require.NoError(t, err)

	got, err := env.repo.GetUserByID(context.Background(), record.ID)

	assert.NoError(t, err)
	assert.Equal(t, record.ID, got.ID)
	assert.Equal(t, "getbyid@example.com", got.Email.String())
	assert.Equal(t, "hashed-password", got.PasswordHash.String())
}

func TestRepository_GetUserByID_NotFound(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	_, err := env.repo.GetUserByID(context.Background(), 99999)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_DeleteUserByID_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "deleteuser@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "deleteuser@example.com").First(&userRec).Error
	require.NoError(t, err)

	err = env.repo.DeleteUserByID(context.Background(), userRec.ID)
	assert.NoError(t, err)

	// Verify deletion
	var count int64
	err = env.db.Model(&userRecord{}).Where("id = ?", userRec.ID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRepository_DeleteUserByID_NotFound(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	err := env.repo.DeleteUserByID(context.Background(), 99999)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
