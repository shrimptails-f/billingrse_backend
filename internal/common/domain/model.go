package domain

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// FetchedEmailDTO は取得直後のメールを表す共通DTOです
type FetchedEmailDTO struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Date    time.Time `json:"date"`
	Body    string    `json:"body"`
}

// ExtractSenderName は From フィールドから送信者名を抽出します
func (b FetchedEmailDTO) ExtractSenderName() string {
	if idx := strings.Index(b.From, "<"); idx > 0 {
		return strings.TrimSpace(b.From[:idx])
	}
	return b.From
}

// ExtractEmailAddress は From フィールドからメールアドレスを抽出します
func (b FetchedEmailDTO) ExtractEmailAddress() string {
	start := strings.Index(b.From, "<")
	end := strings.Index(b.From, ">")
	if start >= 0 && end > start {
		return b.From[start+1 : end]
	}
	return b.From
}

// AnalysisResult は全メール共通の基本情報を表すドメインモデルです
type AnalysisResult struct {
	MailCategory        string   `json:"メール区分"`
	ProjectTitle        string   `json:"案件名"`
	StartPeriod         []string `json:"開始時期"`
	EndPeriod           string   `json:"終了時期"`
	WorkLocation        string   `json:"勤務場所"`
	PriceFrom           *int     `json:"単価FROM"`
	PriceTo             *int     `json:"単価TO"`
	Languages           []string `json:"言語"`
	Frameworks          []string `json:"フレームワーク"`
	Positions           []string `json:"ポジション"`
	WorkTypes           []string `json:"業務"`
	RequiredSkillsMust  []string `json:"求めるスキル MUST"`
	RequiredSkillsWant  []string `json:"求めるスキル WANT"`
	RemoteWorkCategory  *string  `json:"リモートワーク区分"`
	RemoteWorkFrequency *string  `json:"リモートワークの頻度"`
}

// DecryptAESGCM decrypts AES-GCM ciphertext using the provided key.
// ciphertext is expected to be nonce||ciphertext||tag, matching common seal output.
func DecryptAESGCM(key []byte, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// Agent represents an agent connection configuration for a user
type Agent struct {
	ID                 uint
	UserID             uint
	Type               string
	KeyVersion         int16
	Token              []byte
	TokenDigest        []byte
	RefreshToken       []byte
	RefreshTokenDigest []byte
	ExpiresAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// DecryptToken decrypts the stored token using the provided key.
func (a *Agent) DecryptToken(key []byte) (string, error) {
	if len(a.Token) == 0 {
		return "", fmt.Errorf("agent token is empty")
	}
	return DecryptAESGCM(key, a.Token)
}

// DecryptRefreshToken decrypts the stored refresh token using the provided key.
func (a *Agent) DecryptRefreshToken(key []byte) (string, error) {
	if len(a.RefreshToken) == 0 {
		return "", fmt.Errorf("agent refresh token is empty")
	}
	return DecryptAESGCM(key, a.RefreshToken)
}

// AgentType represents supported agent types
type AgentType string

const (
	AgentTypeOpenAI AgentType = "OpenAI"
)

// ValidAgentTypes returns a list of valid agent types
func ValidAgentTypes() []AgentType {
	return []AgentType{
		AgentTypeOpenAI,
	}
}

// IsValidAgentType checks if the given type is valid
func IsValidAgentType(t string) bool {
	for _, validType := range ValidAgentTypes() {
		if string(validType) == t {
			return true
		}
	}
	return false
}

// EncodeBase64 encodes data to base64 string (standard encoding).
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 string to bytes (standard encoding).
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
