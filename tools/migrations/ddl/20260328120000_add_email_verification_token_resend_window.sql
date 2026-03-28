ALTER TABLE `email_verification_tokens`
  ADD COLUMN `resend_window_started_at` datetime(3) NULL AFTER `created_at`,
  ADD COLUMN `resend_count` int unsigned NOT NULL DEFAULT 0 AFTER `resend_window_started_at`;
