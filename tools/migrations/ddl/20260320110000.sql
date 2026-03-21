-- Tighten o_auth_state uniqueness now that it becomes the canonical pending-state lookup key
ALTER TABLE `email_credentials`
  DROP INDEX `idx_email_credentials_o_auth_state`,
  ADD UNIQUE INDEX `uni_email_credentials_o_auth_state` (`o_auth_state`);

-- Drop the obsolete dedicated pending-state table
DROP TABLE `oauth_pending_states`;
