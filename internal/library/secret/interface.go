package secret

import "context"

// Client は AWS Secrets Manager からシークレットを取得するためのインターフェースです。
// 1つのシークレット内に複数キーがある場合は JSON オブジェクトとして登録し、map で受け取ります。
type Client interface {
	// GetValue は指定されたキーの値を返します。
	GetValue(ctx context.Context, key string) (string, error)
}
