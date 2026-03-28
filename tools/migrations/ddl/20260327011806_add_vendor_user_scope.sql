-- Scope "vendors" and "vendor_aliases" by user and enforce per-user uniqueness
ALTER TABLE `vendors`
  ADD COLUMN `user_id` bigint unsigned NOT NULL AFTER `id`,
  DROP INDEX `uni_vendors_normalized_name`,
  ADD UNIQUE INDEX `uni_vendors_user_normalized_name` (`user_id`, `normalized_name`);

ALTER TABLE `vendor_aliases`
  ADD COLUMN `user_id` bigint unsigned NOT NULL AFTER `id`,
  DROP INDEX `idx_vendor_aliases_type_normalized`,
  DROP INDEX `uni_vendor_aliases_vendor_type_value`,
  ADD INDEX `idx_vendor_aliases_user_type_normalized` (`user_id`, `alias_type`, `normalized_value`),
  ADD UNIQUE INDEX `uni_vendor_aliases_user_type_value` (`user_id`, `alias_type`, `normalized_value`);
