ALTER TABLE `billings`
  ADD COLUMN `billing_summary_date` datetime(3) NULL AFTER `billing_date`;

ALTER TABLE `billings`
  MODIFY COLUMN `billing_summary_date` datetime(3) NOT NULL;

ALTER TABLE `billings`
  ADD INDEX `idx_billings_user_summary_date_id` (`user_id`, `billing_summary_date`, `id`),
  ADD INDEX `idx_billings_user_currency_summary_date` (`user_id`, `currency`, `billing_summary_date`),
  ADD INDEX `idx_billings_user_email_id` (`user_id`, `email_id`);
