-- Create "vendors" and "vendor_aliases" tables for deterministic vendor resolution
CREATE TABLE `vendors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `normalized_name` varchar(255) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_vendors_normalized_name` (`normalized_name`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE TABLE `vendor_aliases` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `vendor_id` bigint unsigned NOT NULL,
  `alias_type` varchar(50) NOT NULL,
  `alias_value` text NOT NULL,
  `normalized_value` varchar(255) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_vendor_aliases_vendor_id` (`vendor_id`),
  INDEX `idx_vendor_aliases_type_normalized` (`alias_type`, `normalized_value`),
  UNIQUE INDEX `uni_vendor_aliases_vendor_type_value` (`vendor_id`, `alias_type`, `normalized_value`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
