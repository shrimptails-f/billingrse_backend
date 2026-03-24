-- Create "billings" table for persisted billing aggregates
CREATE TABLE `billings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `vendor_id` bigint unsigned NOT NULL,
  `email_id` bigint unsigned NOT NULL,
  `billing_number` varchar(255) NOT NULL,
  `invoice_number` varchar(14) NULL,
  `amount` decimal(18,3) NOT NULL,
  `currency` char(3) NOT NULL,
  `billing_date` datetime(3) NULL,
  `payment_cycle` varchar(32) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_billings_user_vendor_number` (`user_id`, `vendor_id`, `billing_number`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
