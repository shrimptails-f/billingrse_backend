//go:build integration
// +build integration

package openai

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"business/internal/library/logger"
)

// 注意実際にAPIを呼ぶので利用料金がかかります。
func TestChat_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY is not set")
	}
	limiter := &noOpLimiter{}
	prompt := BuildParsedEmailPrompt("請求メール", "billing@example.com", time.Now().UTC(), getEmailBody())

	c := New(apiKey, limiter, logger.NewNop())

	raw, err := c.Chat(context.Background(), prompt)
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}

	fmt.Printf("raw response:\n%s\n", raw)
}

func getEmailBody() string {
	return `■案件名:PHP Go アプリケーション開発
■場所:大手町
■勤務時間:9:00-18:00
■担当: バックエンド インフラエンジニア
■開始時期: 4月 or 5月
■終了時期: ～長期
■募集:2名
■フレームワーク:Laravel Echo
■必須スキル:
・Gotの開発経験3年
・PHP※年数問わず
・MySQL及びPostgreSQLの経験
■尚可スキル
・ElasticSearch(データソースとして使用)
■単価:65~70万円
■精算:150~200ｈ
■面談:WEB1回(上位同席）)
■リモート リモート可 週３回
■備考:`
}

type noOpLimiter struct{}

func (n *noOpLimiter) Wait(ctx context.Context) error {
	return nil
}
