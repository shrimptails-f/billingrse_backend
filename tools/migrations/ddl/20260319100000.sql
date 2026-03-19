-- Create "oauth_pending_states" table
CREATE TABLE `oauth_pending_states` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `state` varchar(255) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `consumed_at` datetime(3) NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_oauth_pending_states_state` (`state`),
  INDEX `idx_oauth_pending_states_user_id` (`user_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Modify "email_credentials" table: add gmail_address column
ALTER TABLE `email_credentials`
  ADD COLUMN `gmail_address` varchar(255) NOT NULL DEFAULT '' AFTER `type`;

-- Drop old unique index on (user_id, type) and add new one on (user_id, type, gmail_address)
ALTER TABLE `email_credentials`
  DROP INDEX `idx_email_credentials_user_type`,
  ADD UNIQUE INDEX `idx_email_credentials_user_type_gmail` (`user_id`, `type`, `gmail_address`);
