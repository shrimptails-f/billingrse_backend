// このファイルは「repository wrapper は追わない」ことの確認用です。
// wrapper 自体は current package にあるが、その内側の interface 呼び出し実体は追跡しない想定です。
package nplusone_negative_wrapper

import "context"

type UserRepository interface {
	FindByID(ctx context.Context, id int) error
}

type repositoryWrapper struct {
	repository UserRepository
}

func (w *repositoryWrapper) FindByID(ctx context.Context, id int) error {
	return w.repository.FindByID(ctx, id)
}

func notTracked(ctx context.Context, w *repositoryWrapper, ids []int) error {
	for _, id := range ids {
		if err := w.FindByID(ctx, id); err != nil {
			return err
		}
	}

	return nil
}
