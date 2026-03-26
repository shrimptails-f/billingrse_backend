-- Create "billing_line_items" table for billing detail rows
CREATE TABLE `billing_line_items` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `billing_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `position` int NOT NULL,
  `product_name_raw` text NULL,
  `product_name_display` varchar(255) NULL,
  `amount` decimal(18,3) NULL,
  `currency` char(3) NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_billing_line_items_billing_position` (`billing_id`, `position`),
  INDEX `idx_billing_line_items_user_billing` (`user_id`, `billing_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
