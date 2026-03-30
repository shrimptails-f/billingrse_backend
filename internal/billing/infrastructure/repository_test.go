package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"context"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type billingRepoTestEnv struct {
	repo   *BillingRepository
	db     *gorm.DB
	nowUTC time.Time
	clean  func() error
}

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

func newBillingRepoTestEnv(t *testing.T) *billingRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfBillingRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&billingSourceEmailRecord{}, &billingRecord{}, &billingLineItemRecord{}))
	nowUTC := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	return &billingRepoTestEnv{
		repo:   NewBillingRepository(mysqlConn.DB, &billingRepoFixedClock{now: nowUTC}, logger.NewNop()),
		db:     mysqlConn.DB,
		nowUTC: nowUTC,
		clean:  cleanup,
	}
}

func seedBillingSourceEmail(t *testing.T, db *gorm.DB, id, userID uint, receivedAt time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&billingSourceEmailRecord{
		ID:         id,
		UserID:     userID,
		ReceivedAt: receivedAt,
	}).Error)
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

func TestBillingRepository_SaveIfAbsent_CreatedAndDuplicate(t *testing.T) {
	t.Parallel()

	env := newBillingRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	invoiceNumber := "t1234567890123"
	billingDate := time.Date(2026, 3, 24, 8, 0, 0, 0, time.UTC)
	sourceReceivedAt := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	productNameDisplay := "Example Product"
	seedBillingSourceEmail(t, env.db, 3, 1, sourceReceivedAt)
	billing, err := commondomain.NewBilling(
		1,
		2,
		3,
		" INV-001 ",
		&invoiceNumber,
		1200.5,
		" jpy ",
		&billingDate,
		" recurring ",
		&productNameDisplay,
		[]commondomain.BillingLineItemInput{
			{
				ProductNameRaw:     stringPtr("Example Product Full Name"),
				ProductNameDisplay: stringPtr("Example Product"),
				Amount:             float64Ptr(1200.5),
				Currency:           stringPtr("jpy"),
			},
			{
				ProductNameDisplay: stringPtr("Tax"),
				Amount:             float64Ptr(100),
				Currency:           stringPtr("JPY"),
			},
		},
	)
	require.NoError(t, err)

	first, err := env.repo.SaveIfAbsent(ctx, billing)
	require.NoError(t, err)
	require.False(t, first.Duplicate)
	require.NotZero(t, first.BillingID)

	second, err := env.repo.SaveIfAbsent(ctx, billing)
	require.NoError(t, err)
	require.True(t, second.Duplicate)
	require.Equal(t, first.BillingID, second.BillingID)

	var stored billingRecord
	require.NoError(t, env.db.WithContext(ctx).First(&stored, first.BillingID).Error)
	require.Equal(t, uint(1), stored.UserID)
	require.Equal(t, uint(2), stored.VendorID)
	require.Equal(t, uint(3), stored.EmailID)
	require.NotNil(t, stored.ProductNameDisplay)
	require.Equal(t, "Example Product", *stored.ProductNameDisplay)
	require.Equal(t, "INV-001", stored.BillingNumber)
	require.NotNil(t, stored.InvoiceNumber)
	require.Equal(t, "T1234567890123", *stored.InvoiceNumber)
	require.NotNil(t, stored.BillingDate)
	require.True(t, stored.BillingDate.Equal(billingDate))
	require.True(t, stored.BillingSummaryDate.Equal(billingDate))
	require.Equal(t, "recurring", stored.PaymentCycle)
	require.True(t, stored.CreatedAt.Equal(env.nowUTC))
	require.True(t, stored.UpdatedAt.Equal(env.nowUTC))

	var count int64
	require.NoError(t, env.db.WithContext(ctx).Model(&billingRecord{}).Count(&count).Error)
	require.EqualValues(t, 1, count)

	var storedLineItems []billingLineItemRecord
	require.NoError(t, env.db.WithContext(ctx).Order("position asc").Find(&storedLineItems).Error)
	require.Len(t, storedLineItems, 2)
	require.Equal(t, first.BillingID, storedLineItems[0].BillingID)
	require.Equal(t, uint(1), storedLineItems[0].UserID)
	require.Equal(t, 0, storedLineItems[0].Position)
	require.NotNil(t, storedLineItems[0].ProductNameDisplay)
	require.Equal(t, "Example Product", *storedLineItems[0].ProductNameDisplay)
	require.NotNil(t, storedLineItems[0].Amount)
	require.True(t, storedLineItems[0].Amount.Equal(decimal.RequireFromString("1200.5")))
	require.NotNil(t, storedLineItems[0].Currency)
	require.Equal(t, "JPY", *storedLineItems[0].Currency)

	require.Equal(t, 1, storedLineItems[1].Position)
	require.NotNil(t, storedLineItems[1].ProductNameDisplay)
	require.Equal(t, "Tax", *storedLineItems[1].ProductNameDisplay)
	require.NotNil(t, storedLineItems[1].Amount)
	require.True(t, storedLineItems[1].Amount.Equal(decimal.RequireFromString("100")))
}

func TestBillingRepository_SaveIfAbsent_ConcurrentDuplicate(t *testing.T) {
	t.Parallel()

	env := newBillingRepoTestEnv(t)
	defer env.clean()

	sourceReceivedAt := time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC)
	seedBillingSourceEmail(t, env.db, 3, 1, sourceReceivedAt)

	billing, err := commondomain.NewBilling(
		1,
		2,
		3,
		"INV-999",
		nil,
		10,
		"USD",
		nil,
		"one_time",
		nil,
		[]commondomain.BillingLineItemInput{
			{
				ProductNameDisplay: stringPtr("Concurrent Item"),
				Amount:             float64Ptr(10),
				Currency:           stringPtr("usd"),
			},
		},
	)
	require.NoError(t, err)

	const workers = 5
	results := make(chan bool, workers)
	errorsCh := make(chan error, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		//nolint:nplusonecheck // This test intentionally exercises concurrent DB writes in looped goroutines.
		go func() {
			defer wg.Done()
			result, err := env.repo.SaveIfAbsent(context.Background(), billing)
			if err != nil {
				errorsCh <- err
				return
			}
			results <- result.Duplicate
		}()
	}
	wg.Wait()
	close(results)
	close(errorsCh)

	for err := range errorsCh {
		require.NoError(t, err)
	}

	createdCount := 0
	duplicateCount := 0
	for duplicate := range results {
		if duplicate {
			duplicateCount++
			continue
		}
		createdCount++
	}

	require.Equal(t, 1, createdCount)
	require.Equal(t, workers-1, duplicateCount)

	var count int64
	require.NoError(t, env.db.WithContext(context.Background()).Model(&billingRecord{}).Count(&count).Error)
	require.EqualValues(t, 1, count)

	var stored billingRecord
	require.NoError(t, env.db.WithContext(context.Background()).First(&stored).Error)
	require.Nil(t, stored.BillingDate)
	require.True(t, stored.BillingSummaryDate.Equal(sourceReceivedAt))

	var lineItemCount int64
	require.NoError(t, env.db.WithContext(context.Background()).Model(&billingLineItemRecord{}).Count(&lineItemCount).Error)
	require.EqualValues(t, 1, lineItemCount)
}

func TestBillingRepository_SaveIfAbsent_RollsBackWhenLineItemCreateFails(t *testing.T) {
	t.Parallel()

	env := newBillingRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	sourceReceivedAt := time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC)
	seedBillingSourceEmail(t, env.db, 3, 1, sourceReceivedAt)

	billing, err := commondomain.NewBilling(
		1,
		2,
		3,
		"INV-ERR",
		nil,
		10,
		"JPY",
		nil,
		"one_time",
		stringPtr("Broken Item"),
		[]commondomain.BillingLineItemInput{
			{
				ProductNameDisplay: stringPtr("Broken Item"),
				Amount:             float64Ptr(math.NaN()),
				Currency:           stringPtr("JPY"),
			},
		},
	)
	require.NoError(t, err)

	_, err = env.repo.SaveIfAbsent(ctx, billing)
	require.Error(t, err)

	var billingCount int64
	require.NoError(t, env.db.WithContext(ctx).Model(&billingRecord{}).Count(&billingCount).Error)
	require.EqualValues(t, 0, billingCount)

	var lineItemCount int64
	require.NoError(t, env.db.WithContext(ctx).Model(&billingLineItemRecord{}).Count(&lineItemCount).Error)
	require.EqualValues(t, 0, lineItemCount)
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
