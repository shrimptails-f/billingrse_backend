package seeders

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	model "business/tools/migrations/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type seededUser struct {
	ID    uint
	Email string
}

type seededEmail struct {
	ID                uint
	UserID            uint
	ExternalMessageID string
}

type seededBilling struct {
	ID            uint
	UserID        uint
	VendorID      uint
	BillingNumber string
}

type vendorSeed struct {
	Name           string
	NormalizedName string
	Aliases        []vendorAliasSeed
}

type vendorAliasSeed struct {
	AliasType       string
	AliasValue      string
	NormalizedValue string
}

type emailSeed struct {
	UserEmail         string
	Provider          string
	AccountIdentifier string
	ExternalMessageID string
	Subject           string
	FromRaw           string
	To                []string
	ReceivedAt        time.Time
	CreatedRunID      *string
}

type billingSeed struct {
	UserEmail          string
	VendorNormalized   string
	EmailExternalID    string
	ProductNameDisplay *string
	BillingNumber      string
	InvoiceNumber      *string
	Amount             string
	Currency           string
	BillingDate        *time.Time
	PaymentCycle       string
}

// CreateBillingSamples は Billing 一覧 API を試すための関連サンプルデータを投入する。
func CreateBillingSamples(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("gorm db is not configured")
	}

	now := seedReferenceTime()
	usersByEmail, err := loadSeedUsersByEmail(tx, []string{
		"admin@example.com",
		"test@example.com",
		"user@example.com",
	})
	if err != nil {
		return err
	}

	vendorSeeds := []vendorSeed{
		{
			Name:           "AWS",
			NormalizedName: "aws",
			Aliases: []vendorAliasSeed{
				{AliasType: "name_exact", AliasValue: "AWS", NormalizedValue: "aws"},
				{AliasType: "sender_domain", AliasValue: "aws.amazon.com", NormalizedValue: "aws.amazon.com"},
			},
		},
		{
			Name:           "Google Workspace",
			NormalizedName: "google workspace",
			Aliases: []vendorAliasSeed{
				{AliasType: "name_exact", AliasValue: "Google Workspace", NormalizedValue: "google workspace"},
				{AliasType: "sender_domain", AliasValue: "google.com", NormalizedValue: "google.com"},
			},
		},
		{
			Name:           "Notion",
			NormalizedName: "notion",
			Aliases: []vendorAliasSeed{
				{AliasType: "name_exact", AliasValue: "Notion", NormalizedValue: "notion"},
				{AliasType: "sender_domain", AliasValue: "mail.notion.so", NormalizedValue: "mail.notion.so"},
			},
		},
		{
			Name:           "Slack",
			NormalizedName: "slack",
			Aliases: []vendorAliasSeed{
				{AliasType: "name_exact", AliasValue: "Slack", NormalizedValue: "slack"},
				{AliasType: "sender_domain", AliasValue: "slack.com", NormalizedValue: "slack.com"},
			},
		},
		{
			Name:           "OpenAI",
			NormalizedName: "openai",
			Aliases: []vendorAliasSeed{
				{AliasType: "name_exact", AliasValue: "OpenAI", NormalizedValue: "openai"},
				{AliasType: "sender_domain", AliasValue: "openai.com", NormalizedValue: "openai.com"},
			},
		},
	}

	vendors := make([]model.Vendor, 0, len(vendorSeeds))
	for _, seed := range vendorSeeds {
		vendors = append(vendors, model.Vendor{
			Name:           seed.Name,
			NormalizedName: seed.NormalizedName,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&vendors).Error; err != nil {
		return fmt.Errorf("failed to create vendors: %w", err)
	}

	vendorsByNormalizedName, err := loadSeedVendorsByNormalizedName(tx, vendorSeeds)
	if err != nil {
		return err
	}

	aliases := make([]model.VendorAlias, 0, len(vendorSeeds)*2)
	for _, seed := range vendorSeeds {
		vendorID := vendorsByNormalizedName[seed.NormalizedName].ID
		for _, alias := range seed.Aliases {
			aliases = append(aliases, model.VendorAlias{
				VendorID:        vendorID,
				AliasType:       alias.AliasType,
				AliasValue:      alias.AliasValue,
				NormalizedValue: alias.NormalizedValue,
				CreatedAt:       now,
				UpdatedAt:       now,
			})
		}
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&aliases).Error; err != nil {
		return fmt.Errorf("failed to create vendor aliases: %w", err)
	}

	adminRunID := "seed-admin-billing"
	testRunID := "seed-test-billing"
	userRunID := "seed-user-billing"

	emailSeeds := []emailSeed{
		{
			UserEmail:         "test@example.com",
			Provider:          "gmail",
			AccountIdentifier: "billing-demo@test.example.com",
			ExternalMessageID: "seed-test-aws-001",
			Subject:           "[AWS] March 2026 invoice",
			FromRaw:           "AWS Billing <no-reply@aws.amazon.com>",
			To:                []string{"test@example.com"},
			ReceivedAt:        time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
			CreatedRunID:      &testRunID,
		},
		{
			UserEmail:         "test@example.com",
			Provider:          "gmail",
			AccountIdentifier: "billing-demo@test.example.com",
			ExternalMessageID: "seed-test-aws-002",
			Subject:           "[AWS] usage report without billing date",
			FromRaw:           "AWS Billing <no-reply@aws.amazon.com>",
			To:                []string{"test@example.com"},
			ReceivedAt:        time.Date(2026, 3, 18, 11, 45, 0, 0, time.UTC),
			CreatedRunID:      &testRunID,
		},
		{
			UserEmail:         "test@example.com",
			Provider:          "gmail",
			AccountIdentifier: "billing-demo@test.example.com",
			ExternalMessageID: "seed-test-google-001",
			Subject:           "Google Workspace invoice",
			FromRaw:           "Google Workspace <billing-noreply@google.com>",
			To:                []string{"test@example.com"},
			ReceivedAt:        time.Date(2026, 3, 25, 9, 30, 0, 0, time.UTC),
			CreatedRunID:      &testRunID,
		},
		{
			UserEmail:         "test@example.com",
			Provider:          "gmail",
			AccountIdentifier: "billing-demo@test.example.com",
			ExternalMessageID: "seed-test-notion-001",
			Subject:           "Notion plan receipt",
			FromRaw:           "Notion <team@mail.notion.so>",
			To:                []string{"test@example.com"},
			ReceivedAt:        time.Date(2026, 3, 10, 3, 15, 0, 0, time.UTC),
			CreatedRunID:      &testRunID,
		},
		{
			UserEmail:         "test@example.com",
			Provider:          "gmail",
			AccountIdentifier: "billing-demo@test.example.com",
			ExternalMessageID: "seed-test-slack-001",
			Subject:           "Slack invoice for March",
			FromRaw:           "Slack <billing@slack.com>",
			To:                []string{"test@example.com"},
			ReceivedAt:        time.Date(2026, 3, 12, 6, 0, 0, 0, time.UTC),
			CreatedRunID:      &testRunID,
		},
		{
			UserEmail:         "admin@example.com",
			Provider:          "gmail",
			AccountIdentifier: "admin@example.com",
			ExternalMessageID: "seed-admin-openai-001",
			Subject:           "OpenAI API invoice",
			FromRaw:           "OpenAI Billing <billing@openai.com>",
			To:                []string{"admin@example.com"},
			ReceivedAt:        time.Date(2026, 3, 22, 8, 0, 0, 0, time.UTC),
			CreatedRunID:      &adminRunID,
		},
		{
			UserEmail:         "user@example.com",
			Provider:          "gmail",
			AccountIdentifier: "user@example.com",
			ExternalMessageID: "seed-user-aws-001",
			Subject:           "AWS invoice for another user",
			FromRaw:           "AWS Billing <no-reply@aws.amazon.com>",
			To:                []string{"user@example.com"},
			ReceivedAt:        time.Date(2026, 3, 16, 7, 0, 0, 0, time.UTC),
			CreatedRunID:      &userRunID,
		},
	}

	emails := make([]model.Email, 0, len(emailSeeds))
	for _, seed := range emailSeeds {
		user, ok := usersByEmail[seed.UserEmail]
		if !ok {
			return fmt.Errorf("seed user not found: %s", seed.UserEmail)
		}

		toJSON, err := marshalSeedRecipients(seed.To)
		if err != nil {
			return err
		}

		emails = append(emails, model.Email{
			UserID:            user.ID,
			Provider:          seed.Provider,
			AccountIdentifier: seed.AccountIdentifier,
			ExternalMessageID: seed.ExternalMessageID,
			Subject:           seed.Subject,
			FromRaw:           seed.FromRaw,
			ToJSON:            toJSON,
			BodyDigest:        hashSeedBody(seed.Subject, seed.ExternalMessageID),
			ReceivedAt:        seed.ReceivedAt,
			CreatedRunID:      seed.CreatedRunID,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&emails).Error; err != nil {
		return fmt.Errorf("failed to create emails: %w", err)
	}

	emailsByKey, err := loadSeedEmailsByUserAndExternalID(tx, emailSeeds, usersByEmail)
	if err != nil {
		return err
	}
	seedEmailReceivedAtByKey := make(map[string]time.Time, len(emailSeeds))
	for _, seed := range emailSeeds {
		user := usersByEmail[seed.UserEmail]
		seedEmailReceivedAtByKey[buildSeedEmailKey(user.ID, seed.ExternalMessageID)] = seed.ReceivedAt.UTC()
	}

	billingDateAWSPrimary := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	billingDateGoogle := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	billingDateNotion := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	billingDateSlack := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	billingDateOpenAI := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	billingDateOtherUser := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	billingSeeds := []billingSeed{
		{
			UserEmail:          "test@example.com",
			VendorNormalized:   "aws",
			EmailExternalID:    "seed-test-aws-001",
			ProductNameDisplay: stringPtr("AWS Support Enterprise"),
			BillingNumber:      "INV-AWS-2026-03-001",
			InvoiceNumber:      stringPtr("T1234567890123"),
			Amount:             "12345.678",
			Currency:           "JPY",
			BillingDate:        &billingDateAWSPrimary,
			PaymentCycle:       "recurring",
		},
		{
			UserEmail:          "test@example.com",
			VendorNormalized:   "aws",
			EmailExternalID:    "seed-test-aws-002",
			ProductNameDisplay: stringPtr("AWS Support Business"),
			BillingNumber:      "INV-AWS-2026-03-002",
			Amount:             "800.000",
			Currency:           "JPY",
			BillingDate:        nil,
			PaymentCycle:       "recurring",
		},
		{
			UserEmail:          "test@example.com",
			VendorNormalized:   "google workspace",
			EmailExternalID:    "seed-test-google-001",
			ProductNameDisplay: stringPtr("Google Workspace Business Standard"),
			BillingNumber:      "INV-GWS-2026-03-001",
			Amount:             "99.990",
			Currency:           "USD",
			BillingDate:        &billingDateGoogle,
			PaymentCycle:       "recurring",
		},
		{
			UserEmail:          "test@example.com",
			VendorNormalized:   "notion",
			EmailExternalID:    "seed-test-notion-001",
			ProductNameDisplay: stringPtr("Notion Plus"),
			BillingNumber:      "INV-NOTION-2026-03-001",
			Amount:             "18.000",
			Currency:           "USD",
			BillingDate:        &billingDateNotion,
			PaymentCycle:       "one_time",
		},
		{
			UserEmail:          "test@example.com",
			VendorNormalized:   "slack",
			EmailExternalID:    "seed-test-slack-001",
			ProductNameDisplay: stringPtr("Slack Pro"),
			BillingNumber:      "INV-SLACK-2026-03-001",
			Amount:             "24.500",
			Currency:           "USD",
			BillingDate:        &billingDateSlack,
			PaymentCycle:       "recurring",
		},
		{
			UserEmail:          "admin@example.com",
			VendorNormalized:   "openai",
			EmailExternalID:    "seed-admin-openai-001",
			ProductNameDisplay: stringPtr("OpenAI API Platform"),
			BillingNumber:      "INV-OPENAI-2026-03-001",
			Amount:             "250.000",
			Currency:           "USD",
			BillingDate:        &billingDateOpenAI,
			PaymentCycle:       "recurring",
		},
		{
			UserEmail:          "user@example.com",
			VendorNormalized:   "aws",
			EmailExternalID:    "seed-user-aws-001",
			ProductNameDisplay: stringPtr("AWS Activate Credits"),
			BillingNumber:      "INV-AWS-USER-2026-03-001",
			Amount:             "5000.000",
			Currency:           "JPY",
			BillingDate:        &billingDateOtherUser,
			PaymentCycle:       "one_time",
		},
	}

	hasAmountColumn := tx.Migrator().HasColumn("billings", "amount")
	hasCurrencyColumn := tx.Migrator().HasColumn("billings", "currency")
	if hasAmountColumn != hasCurrencyColumn {
		return fmt.Errorf("billings schema mismatch: amount and currency columns must be both present or both absent")
	}

	billingRows := make([]map[string]any, 0, len(billingSeeds))
	for _, seed := range billingSeeds {
		user, ok := usersByEmail[seed.UserEmail]
		if !ok {
			return fmt.Errorf("seed user not found for billing: %s", seed.UserEmail)
		}

		vendor, ok := vendorsByNormalizedName[seed.VendorNormalized]
		if !ok {
			return fmt.Errorf("seed vendor not found: %s", seed.VendorNormalized)
		}

		emailKey := buildSeedEmailKey(user.ID, seed.EmailExternalID)
		email, ok := emailsByKey[emailKey]
		if !ok {
			return fmt.Errorf("seed email not found: user=%s external_message_id=%s", seed.UserEmail, seed.EmailExternalID)
		}

		row := map[string]any{
			"user_id":              user.ID,
			"vendor_id":            vendor.ID,
			"email_id":             email.ID,
			"product_name_display": cloneSeedString(seed.ProductNameDisplay),
			"billing_number":       strings.TrimSpace(seed.BillingNumber),
			"invoice_number":       cloneSeedString(seed.InvoiceNumber),
			"billing_date":         cloneSeedTime(seed.BillingDate),
			"billing_summary_date": resolveSeedBillingSummaryDate(seed.BillingDate, seedEmailReceivedAtByKey[emailKey]),
			"payment_cycle":        strings.TrimSpace(seed.PaymentCycle),
			"created_at":           now,
			"updated_at":           now,
		}
		if hasAmountColumn {
			amountDecimal, err := decimal.NewFromString(strings.TrimSpace(seed.Amount))
			if err != nil {
				return fmt.Errorf("invalid seed amount for billing_number=%s: %w", seed.BillingNumber, err)
			}
			currency := strings.TrimSpace(strings.ToUpper(seed.Currency))
			row["amount"] = amountDecimal
			row["currency"] = currency
		}

		billingRows = append(billingRows, row)
	}

	if err := tx.Table("billings").Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&billingRows).Error; err != nil {
		return fmt.Errorf("failed to create billings: %w", err)
	}

	billingsByKey, err := loadSeedBillingsByIdentity(tx, billingSeeds, usersByEmail, vendorsByNormalizedName)
	if err != nil {
		return err
	}

	lineItems := make([]model.BillingLineItem, 0, len(billingSeeds))
	for _, seed := range billingSeeds {
		user := usersByEmail[seed.UserEmail]
		vendor := vendorsByNormalizedName[seed.VendorNormalized]
		billingKey := buildSeedBillingKey(user.ID, vendor.ID, seed.BillingNumber)
		billing, ok := billingsByKey[billingKey]
		if !ok {
			return fmt.Errorf("seed billing not found for line item: user=%s vendor=%s billing_number=%s", seed.UserEmail, seed.VendorNormalized, seed.BillingNumber)
		}

		amountDecimal, err := decimal.NewFromString(strings.TrimSpace(seed.Amount))
		if err != nil {
			return fmt.Errorf("invalid seed amount for billing_number=%s: %w", seed.BillingNumber, err)
		}

		lineItems = append(lineItems, model.BillingLineItem{
			BillingID:          billing.ID,
			UserID:             user.ID,
			Position:           0,
			ProductNameDisplay: cloneSeedString(seed.ProductNameDisplay),
			Amount:             float64Ptr(amountDecimal.InexactFloat64()),
			Currency:           normalizeSeedCurrency(seed.Currency),
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "billing_id"},
			{Name: "position"},
		},
		DoNothing: true,
	}).Create(&lineItems).Error; err != nil {
		return fmt.Errorf("failed to create billing line items: %w", err)
	}

	return nil
}

func loadSeedUsersByEmail(tx *gorm.DB, emails []string) (map[string]seededUser, error) {
	var users []seededUser
	if err := tx.Table("users").Select("id", "email").Where("email IN ?", emails).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to load seed users: %w", err)
	}

	usersByEmail := make(map[string]seededUser, len(users))
	for _, user := range users {
		usersByEmail[strings.TrimSpace(user.Email)] = user
	}

	for _, email := range emails {
		if _, ok := usersByEmail[email]; !ok {
			return nil, fmt.Errorf("required seed user not found: %s", email)
		}
	}

	return usersByEmail, nil
}

func loadSeedVendorsByNormalizedName(tx *gorm.DB, seeds []vendorSeed) (map[string]model.Vendor, error) {
	normalizedNames := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		normalizedNames = append(normalizedNames, seed.NormalizedName)
	}

	var vendors []model.Vendor
	if err := tx.Where("normalized_name IN ?", normalizedNames).Find(&vendors).Error; err != nil {
		return nil, fmt.Errorf("failed to load seed vendors: %w", err)
	}

	vendorsByNormalizedName := make(map[string]model.Vendor, len(vendors))
	for _, vendor := range vendors {
		vendorsByNormalizedName[strings.TrimSpace(vendor.NormalizedName)] = vendor
	}

	for _, normalizedName := range normalizedNames {
		if _, ok := vendorsByNormalizedName[normalizedName]; !ok {
			return nil, fmt.Errorf("required seed vendor not found: %s", normalizedName)
		}
	}

	return vendorsByNormalizedName, nil
}

func loadSeedEmailsByUserAndExternalID(
	tx *gorm.DB,
	seeds []emailSeed,
	usersByEmail map[string]seededUser,
) (map[string]seededEmail, error) {
	userIDs := make([]uint, 0, len(usersByEmail))
	externalIDs := make([]string, 0, len(seeds))
	for _, user := range usersByEmail {
		userIDs = append(userIDs, user.ID)
	}
	for _, seed := range seeds {
		externalIDs = append(externalIDs, seed.ExternalMessageID)
	}

	var emails []seededEmail
	if err := tx.Table("emails").
		Select("id", "user_id", "external_message_id").
		Where("user_id IN ? AND external_message_id IN ?", userIDs, externalIDs).
		Find(&emails).Error; err != nil {
		return nil, fmt.Errorf("failed to load seed emails: %w", err)
	}

	emailsByKey := make(map[string]seededEmail, len(emails))
	for _, email := range emails {
		emailsByKey[buildSeedEmailKey(email.UserID, email.ExternalMessageID)] = email
	}

	for _, seed := range seeds {
		user := usersByEmail[seed.UserEmail]
		key := buildSeedEmailKey(user.ID, seed.ExternalMessageID)
		if _, ok := emailsByKey[key]; !ok {
			return nil, fmt.Errorf("required seed email not found: user=%s external_message_id=%s", seed.UserEmail, seed.ExternalMessageID)
		}
	}

	return emailsByKey, nil
}

func loadSeedBillingsByIdentity(
	tx *gorm.DB,
	seeds []billingSeed,
	usersByEmail map[string]seededUser,
	vendorsByNormalizedName map[string]model.Vendor,
) (map[string]seededBilling, error) {
	userIDs := make([]uint, 0, len(usersByEmail))
	for _, user := range usersByEmail {
		userIDs = append(userIDs, user.ID)
	}

	vendorIDs := make([]uint, 0, len(vendorsByNormalizedName))
	for _, vendor := range vendorsByNormalizedName {
		vendorIDs = append(vendorIDs, vendor.ID)
	}

	billingNumbers := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		billingNumbers = append(billingNumbers, strings.TrimSpace(seed.BillingNumber))
	}

	var billings []seededBilling
	if err := tx.Table("billings").
		Select("id", "user_id", "vendor_id", "billing_number").
		Where("user_id IN ?", userIDs).
		Where("vendor_id IN ?", vendorIDs).
		Where("billing_number IN ?", billingNumbers).
		Find(&billings).Error; err != nil {
		return nil, fmt.Errorf("failed to load seed billings: %w", err)
	}

	billingsByKey := make(map[string]seededBilling, len(billings))
	for _, billing := range billings {
		key := buildSeedBillingKey(billing.UserID, billing.VendorID, billing.BillingNumber)
		billingsByKey[key] = billing
	}

	for _, seed := range seeds {
		user := usersByEmail[seed.UserEmail]
		vendor := vendorsByNormalizedName[seed.VendorNormalized]
		key := buildSeedBillingKey(user.ID, vendor.ID, seed.BillingNumber)
		if _, ok := billingsByKey[key]; !ok {
			return nil, fmt.Errorf("required seed billing not found: user=%s vendor=%s billing_number=%s", seed.UserEmail, seed.VendorNormalized, seed.BillingNumber)
		}
	}

	return billingsByKey, nil
}

func buildSeedEmailKey(userID uint, externalMessageID string) string {
	return fmt.Sprintf("%d:%s", userID, strings.TrimSpace(externalMessageID))
}

func buildSeedBillingKey(userID uint, vendorID uint, billingNumber string) string {
	return fmt.Sprintf("%d:%d:%s", userID, vendorID, strings.TrimSpace(billingNumber))
}

func marshalSeedRecipients(recipients []string) (string, error) {
	value, err := json.Marshal(recipients)
	if err != nil {
		return "", fmt.Errorf("failed to marshal recipients: %w", err)
	}
	return string(value), nil
}

func hashSeedBody(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func seedReferenceTime() time.Time {
	return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func cloneSeedString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := strings.TrimSpace(*value)
	return &cloned
}

func cloneSeedTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}

func resolveSeedBillingSummaryDate(billingDate *time.Time, fallbackReceivedAt time.Time) time.Time {
	if billingDate != nil {
		return billingDate.UTC()
	}
	return fallbackReceivedAt.UTC()
}

func normalizeSeedCurrency(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	upper := strings.ToUpper(trimmed)
	return &upper
}
