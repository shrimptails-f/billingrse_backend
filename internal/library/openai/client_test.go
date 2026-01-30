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
	prompt := buildParsedEmailPrompt(getEmailBody())

	c := New(apiKey, limiter, logger.NewNop())

	analysisResults, err := c.Chat(context.Background(), prompt)
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}

	for i, item := range analysisResults {
		fmt.Printf("---- 結果 %d ----\n", i+1)
		if item.VendorName != nil {
			fmt.Printf("支払先名: %s\n", *item.VendorName)
		}
		if item.InvoiceNumber != nil {
			fmt.Printf("請求番号: %s\n", *item.InvoiceNumber)
		}
		if item.Amount != nil {
			fmt.Printf("金額: %d\n", *item.Amount)
		}
		if item.Currency != nil {
			fmt.Printf("通貨: %s\n", *item.Currency)
		}
		if item.BillingDate != nil {
			fmt.Printf("請求日: %s\n", item.BillingDate.Format(time.RFC3339))
		}
		if item.PaymentType != nil {
			fmt.Printf("支払いタイプ: %s\n", *item.PaymentType)
		}
		fmt.Printf("抽出日時: %s\n", item.ExtractedAt.Format(time.RFC3339))
	}
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
	return fmt.Sprintf(`以下は請求に関するメール本文です。本文を読み取り、JSON配列のみを出力してください。

ルール:
- 出力はJSON配列のみ（前後に説明文を入れない）
- 要素のキーは vendorName, invoiceNumber, amount, currency, billingDate, paymentType, extractedAt のみ
- 不明な値は null をセット
- billingDate と extractedAt は RFC3339 形式の文字列
- amount は整数
- 複数の請求が含まれる場合は配列に複数要素を入れる

本文:
%s`, body)
}

type noOpLimiter struct{}

func (n *noOpLimiter) Wait(ctx context.Context) error {
	return nil
}
