-- Create "emails" table for idempotent raw fetched email metadata
CREATE TABLE `emails` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `provider` varchar(50) NOT NULL,
  `account_identifier` varchar(255) NOT NULL,
  `external_message_id` varchar(255) NOT NULL,
  `subject` text NOT NULL,
  `from_raw` text NOT NULL,
  `to_json` json NOT NULL,
  `received_at` datetime(3) NOT NULL,
  `created_run_id` varchar(36) NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_emails_user_message` (`user_id`, `external_message_id`),
  INDEX `idx_emails_user_received_at` (`user_id`, `received_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
