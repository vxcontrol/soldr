-- +migrate Up

ALTER TABLE `external_connections` CHANGE COLUMN `type` `type` enum('aggregate','browser','external') NOT NULL;

-- +migrate Down

ALTER TABLE `external_connections` CHANGE COLUMN `type` `type` enum('browser','external') NOT NULL;
