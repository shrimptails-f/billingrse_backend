// このファイルは「interface 経由の呼び出し」ケースのテストデータです。
// application/usecase が repository interface を持ち、
// infrastructure の concrete repository が query を実行する形を見ます。
package nplusone_interface

import (
	"context"
	"database/sql"
)

type user struct {
	ID int
}

type UserRepository interface {
	FindByID(ctx context.Context, id int) (user, error)
}

type UseCase interface {
	Execute(ctx context.Context, ids []int) error
}

type useCase struct {
	repository UserRepository
}

func NewUseCase(repository UserRepository) UseCase {
	return &useCase{repository: repository}
}

// 検知ケース:
// usecase は UserRepository interface しか知らないが、
// 実装は gormUserRepository の 1 つに静的に絞れるため検知される想定です。
func (uc *useCase) Execute(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if _, err := uc.repository.FindByID(ctx, id); err != nil { // want "possible N\\+1 query inside loop"
			return err
		}
	}

	return nil
}

type gormUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *gormUserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) FindByID(ctx context.Context, id int) (user, error) {
	var record user
	_ = r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE id = ?", id)
	return record, nil
}

type ProfileRepository interface {
	LoadByID(ctx context.Context, id int) error
}

type profileListUseCase struct {
	repository ProfileRepository
}

// 検知ケース:
// ProfileRepository は複数実装を持つが、
// sqlProfileRepository 側に query 到達経路があるため保守的に検知する想定です。
func (uc *profileListUseCase) Execute(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := uc.repository.LoadByID(ctx, id); err != nil { // want "possible N\\+1 query inside loop"
			return err
		}
	}

	return nil
}

type sqlProfileRepository struct {
	db *sql.DB
}

func (r *sqlProfileRepository) LoadByID(ctx context.Context, id int) error {
	_ = r.db.QueryRowContext(ctx, "SELECT id FROM profiles WHERE id = ?", id)
	return nil
}

type memoryProfileRepository struct{}

func (r *memoryProfileRepository) LoadByID(context.Context, int) error {
	return nil
}

type AuditRepository interface {
	TouchByID(ctx context.Context, id int) error
}

type auditUseCase struct {
	repository AuditRepository
}

// 非検知ケース:
// AuditRepository は複数実装を持つが、
// どの候補にも query 到達経路がないため検知しない想定です。
func (uc *auditUseCase) Execute(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := uc.repository.TouchByID(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

type auditLogRepository struct{}

func (r *auditLogRepository) TouchByID(context.Context, int) error {
	return nil
}

type noopAuditRepository struct{}

func (r *noopAuditRepository) TouchByID(context.Context, int) error {
	return nil
}
