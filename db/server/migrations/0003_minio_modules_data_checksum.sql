-- +migrate Up

ALTER TABLE `modules`
    ADD COLUMN `files_checksums` JSON NOT NULL
    AFTER `system`;

UPDATE `modules` SET `files_checksums`=(JSON_OBJECT()) WHERE JSON_TYPE(CAST(`files_checksums` AS JSON)) = 'NULL';

-- +migrate Down

ALTER TABLE `modules` DROP COLUMN `files_checksums`;
