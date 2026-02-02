CREATE SCHEMA IF NOT EXISTS `development` DEFAULT CHARACTER SET utf8mb4 ;

-- 'user'@'%' が未作成の場合、ユーザーを作成
CREATE USER IF NOT EXISTS 'user'@'%' IDENTIFIED BY 'password';
CREATE DATABASE `atlas` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
GRANT ALL PRIVILEGES ON `atlas`.* TO 'user'@'%';
FLUSH PRIVILEGES;

-- セッションの文字セットをutf8mb4に設定
SET NAMES 'utf8mb4';