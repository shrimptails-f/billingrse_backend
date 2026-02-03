-- Create "email_credentials" table
CREATE TABLE `email_credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `type` varchar(50) NOT NULL,
  `key_version` smallint NOT NULL DEFAULT 1,
  `access_token` text NOT NULL,
  `access_token_digest` text NOT NULL,
  `refresh_token` text NOT NULL,
  `refresh_token_digest` text NOT NULL,
  `token_expiry` datetime(3) NULL,
  `o_auth_state` varchar(255) NULL,
  `o_auth_state_expires_at` datetime(3) NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_email_credentials_o_auth_state` (`o_auth_state`),
  UNIQUE INDEX `idx_email_credentials_user_type` (`user_id`, `type`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "email_verification_tokens" table
CREATE TABLE `email_verification_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `token` varchar(36) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `consumed_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_token` (`token`),
  UNIQUE INDEX `idx_user_id_unique` (`user_id`),
  UNIQUE INDEX `uni_email_verification_tokens_token` (`token`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "users" table
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `email_verified` bool NOT NULL DEFAULT 0,
  `email_verified_at` datetime(3) NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_users_email` (`email`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
