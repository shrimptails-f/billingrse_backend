package infrastructure

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type billingRepoFixedClock struct {
	now time.Time
}

func (c *billingRepoFixedClock) Now() time.Time {
	return c.now
}

func (c *billingRepoFixedClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func skipIfBillingRepoDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

type billingFixture struct {
	ID                 uint
	UserID             uint
	VendorID           uint
	EmailID            uint
	ProductNameDisplay *string
	BillingNumber      string
	InvoiceNumber      *string
	Amount             decimal.Decimal
	Currency           string
	BillingDate        *time.Time
	BillingSummaryDate time.Time
	PaymentCycle       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func billingRecordsAndLineItemsFromFixtures(fixtures []billingFixture) ([]billingRecord, []billingLineItemRecord) {
	records := make([]billingRecord, 0, len(fixtures))
	lineItems := make([]billingLineItemRecord, 0, len(fixtures))
	for _, fixture := range fixtures {
		records = append(records, billingRecord{
			ID:                 fixture.ID,
			UserID:             fixture.UserID,
			VendorID:           fixture.VendorID,
			EmailID:            fixture.EmailID,
			ProductNameDisplay: cloneOptionalString(fixture.ProductNameDisplay),
			BillingNumber:      strings.TrimSpace(fixture.BillingNumber),
			InvoiceNumber:      cloneOptionalString(fixture.InvoiceNumber),
			BillingDate:        cloneBillingDate(fixture.BillingDate),
			BillingSummaryDate: fixture.BillingSummaryDate.UTC(),
			PaymentCycle:       strings.TrimSpace(fixture.PaymentCycle),
			CreatedAt:          fixture.CreatedAt.UTC(),
			UpdatedAt:          fixture.UpdatedAt.UTC(),
		})

		amount := fixture.Amount.InexactFloat64()
		currencyValue := strings.TrimSpace(strings.ToUpper(fixture.Currency))
		var currency *string
		if currencyValue != "" {
			currency = &currencyValue
		}
		lineItems = append(lineItems, billingLineItemRecord{
			BillingID:          fixture.ID,
			UserID:             fixture.UserID,
			Position:           0,
			ProductNameDisplay: cloneOptionalString(fixture.ProductNameDisplay),
			Amount:             &amount,
			Currency:           currency,
			CreatedAt:          fixture.CreatedAt.UTC(),
			UpdatedAt:          fixture.UpdatedAt.UTC(),
		})
	}

	return records, lineItems
}
