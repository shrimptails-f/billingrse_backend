ALTER TABLE `emails`
  ADD COLUMN `body_digest` varchar(64) NOT NULL AFTER `to_json`;
