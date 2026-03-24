//go:build integration
// +build integration

package openai

import (
	"context"
	"fmt"
	"os"
	"testing"

	"business/internal/library/logger"
)

// 注意実際にAPIを呼ぶので利用料金がかかります。
func TestChat_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY is not set")
	}
	limiter := &noOpLimiter{}
	prompt := buildParsedEmailPrompt(getEmailBody())

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

func buildParsedEmailPrompt(body string) string {
	return fmt.Sprintf(`以下は請求に関するメール本文です。本文を読み取り、JSONオブジェクトのみを出力してください。

ルール:
- 出力はJSONオブジェクトのみ（前後に説明文を入れない）
- トップレベルのキーは parsedEmails のみ
- parsedEmails は配列
- parsedEmails の各要素のキーは productNameRaw, productNameDisplay, vendorName, billingNumber, invoiceNumber, amount, currency, billingDate, paymentCycle のみ
- productNameRaw はメール内の全文
- productNameDisplay は表示用の短い名前。単品なら商品名だけ、セット商品ならセット名
- 不明な値は null をセット
- billingDate は RFC3339 形式の文字列
- amount は小数第3位までの数値
- invoiceNumber は 14 文字以内
- 複数の請求が含まれる場合は parsedEmails に複数要素を入れる

本文:
%s`, body)
}

type noOpLimiter struct{}

func (n *noOpLimiter) Wait(ctx context.Context) error {
	return nil
}
