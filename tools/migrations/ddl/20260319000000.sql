-- Create "auth_refresh_tokens" table
CREATE TABLE `auth_refresh_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `token_digest` varchar(64) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `last_used_at` datetime(3) NULL,
  `revoked_at` datetime(3) NULL,
  `replaced_by_token_id` bigint unsigned NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_auth_refresh_tokens_token_digest` (`token_digest`),
  INDEX `idx_auth_refresh_tokens_user_id` (`user_id`),
  INDEX `idx_auth_refresh_tokens_replaced_by_token_id` (`replaced_by_token_id`),
  INDEX `idx_auth_refresh_tokens_expires_at` (`expires_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
