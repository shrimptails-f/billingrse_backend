ALTER TABLE `billings`
  DROP INDEX `idx_billings_user_currency_summary_date`,
  DROP COLUMN `amount`,
  DROP COLUMN `currency`;
