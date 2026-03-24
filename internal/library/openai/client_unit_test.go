package openai

import (
	"business/internal/library/logger"
	"business/internal/library/retry"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// buildChatCompletionParams が意図したモデルと schema 構成を組み立てることを確認する。
// 送信前の Go オブジェクトに対するテストなので、Structured Outputs の設定漏れを早い段階で検知する。
func TestBuildChatCompletionParams_UsesGPT5MiniStructuredOutputs(t *testing.T) {
	t.Parallel()

	params := buildChatCompletionParams("extract billing information")

	if params.Model != DefaultModel {
		t.Fatalf("unexpected model: %s", params.Model)
	}
	if len(params.Messages) != 1 {
		t.Fatalf("unexpected message count: %d", len(params.Messages))
	}
	if params.ResponseFormat.OfJSONSchema == nil {
		t.Fatal("expected json_schema response format")
	}

	schema := params.ResponseFormat.OfJSONSchema.JSONSchema
	if schema.Name != parsedEmailResponseSchemaName {
		t.Fatalf("unexpected schema name: %s", schema.Name)
	}
	if !schema.Strict.Valid() || !schema.Strict.Value {
		t.Fatalf("expected strict structured outputs, got %+v", schema.Strict)
	}
	if !schema.Description.Valid() || schema.Description.Value == "" {
		t.Fatalf("expected schema description, got %+v", schema.Description)
	}

	root, ok := schema.Schema.(map[string]any)
	if !ok {
		t.Fatalf("unexpected schema type: %T", schema.Schema)
	}
	if got := root["type"]; got != "object" {
		t.Fatalf("unexpected root type: %#v", got)
	}
	if got := root["additionalProperties"]; got != false {
		t.Fatalf("expected root additionalProperties=false, got %#v", got)
	}

	properties, ok := root["properties"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected properties type: %T", root["properties"])
	}

	parsedEmails, ok := properties["parsedEmails"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected parsedEmails type: %T", properties["parsedEmails"])
	}
	if got := parsedEmails["type"]; got != "array" {
		t.Fatalf("unexpected parsedEmails type: %#v", got)
	}

	items, ok := parsedEmails["items"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected items type: %T", parsedEmails["items"])
	}
	if got := items["additionalProperties"]; got != false {
		t.Fatalf("expected additionalProperties=false, got %#v", got)
	}

	itemProperties, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected properties type: %T", items["properties"])
	}
	for _, key := range []string{
		"productNameRaw",
		"productNameDisplay",
		"vendorName",
		"billingNumber",
		"invoiceNumber",
		"amount",
		"currency",
		"billingDate",
		"paymentCycle",
	} {
		if _, exists := itemProperties[key]; !exists {
			t.Fatalf("expected property %q to exist", key)
		}
	}
}

// Chat Completions リクエストとして marshal した最終 JSON に
// response_format=json_schema と strict schema が実際に載ることを確認する。
// SDK の parameter object 組み立てと JSON 化の両方を壊していないかを見る観点。
func TestBuildChatCompletionParams_MarshalJSONIncludesStructuredOutputs(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(buildChatCompletionParams("extract billing information"))
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if got := body["model"]; got != DefaultModel {
		t.Fatalf("unexpected model in payload: %#v", got)
	}

	responseFormat, ok := body["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected response_format type: %T", body["response_format"])
	}
	if got := responseFormat["type"]; got != "json_schema" {
		t.Fatalf("unexpected response_format type: %#v", got)
	}

	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected json_schema type: %T", responseFormat["json_schema"])
	}
	if got := jsonSchema["name"]; got != parsedEmailResponseSchemaName {
		t.Fatalf("unexpected schema name in payload: %#v", got)
	}
	if got := jsonSchema["strict"]; got != true {
		t.Fatalf("unexpected strict flag in payload: %#v", got)
	}
}

func TestChat_RateLimiterGatesEveryHTTPAttempt(t *testing.T) {
	oldBackoff := retry.DefaultBackoff
	retry.DefaultBackoff = []time.Duration{0}
	defer func() {
		retry.DefaultBackoff = oldBackoff
	}()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch atomic.AddInt32(&requestCount, 1) {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":{"message":"server error","type":"server_error","param":"","code":""}}`)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"chatcmpl_test","object":"chat.completion","created":1,"model":"gpt-5-mini","choices":[{"index":0,"finish_reason":"stop","logprobs":{"content":[],"refusal":[]},"message":{"role":"assistant","content":"{\"parsedEmails\":[]}","refusal":""}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		}
	}))
	defer server.Close()

	t.Setenv("OPENAI_BASE_URL", server.URL+"/v1")

	limiter := &countingLimiter{}
	client := New("test-api-key", limiter, logger.NewNop())

	raw, err := client.Chat(context.Background(), "extract billing information")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if raw != `{"parsedEmails":[]}` {
		t.Fatalf("unexpected raw response: %s", raw)
	}
	if got := limiter.Calls(); got != 2 {
		t.Fatalf("expected limiter to gate both attempts, got %d", got)
	}
	if got := int(atomic.LoadInt32(&requestCount)); got != 2 {
		t.Fatalf("expected exactly 2 HTTP requests, got %d", got)
	}
}

func TestChat_LimiterErrorPreventsHTTPRequest(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"chatcmpl_test","object":"chat.completion","created":1,"model":"gpt-5-mini","choices":[{"index":0,"finish_reason":"stop","logprobs":{"content":[],"refusal":[]},"message":{"role":"assistant","content":"{\"parsedEmails\":[]}","refusal":""}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	t.Setenv("OPENAI_BASE_URL", server.URL+"/v1")

	expectedErr := errors.New("limiter unavailable")
	limiter := &countingLimiter{err: expectedErr}
	client := New("test-api-key", limiter, logger.NewNop())

	_, err := client.Chat(context.Background(), "extract billing information")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected limiter error, got %v", err)
	}
	if got := limiter.Calls(); got != 1 {
		t.Fatalf("expected limiter to be called once, got %d", got)
	}
	if got := int(atomic.LoadInt32(&requestCount)); got != 0 {
		t.Fatalf("expected no HTTP request when limiter fails, got %d", got)
	}
}

type countingLimiter struct {
	calls int32
	err   error
}

func (l *countingLimiter) Wait(ctx context.Context) error {
	atomic.AddInt32(&l.calls, 1)
	return l.err
}

func (l *countingLimiter) Calls() int {
	return int(atomic.LoadInt32(&l.calls))
}
