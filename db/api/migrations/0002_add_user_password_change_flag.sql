-- +migrate Up

ALTER TABLE `users` ADD COLUMN
    `password_change_required` BOOL NOT NULL DEFAULT false AFTER `tenant_id`;

-- +migrate Down

ALTER TABLE `users` DROP COLUMN `password_change_required`;