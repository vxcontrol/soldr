-- +migrate Up

CREATE TABLE IF NOT EXISTS `roles`
(
    `id`   int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(50) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=101 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `tenants`
(
    `id`          int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`        varchar(32) NOT NULL,
    `uuid`        varchar(100) DEFAULT NULL,
    `status`      enum('active','blocked') NOT NULL,
    `description` varchar(32) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `hash_idx` (`hash`),
    KEY        `uuid_idx` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `binaries`
(
    `id`          int(10) unsigned NOT NULL AUTO_INCREMENT,
    `tenant_id`   int(10) unsigned NOT NULL,
    `hash`        varchar(32) NOT NULL,
    `type`        enum('vxagent') NOT NULL,
    `info`        json        NOT NULL,
    `ver_major`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.major'))) STORED,
    `ver_minor`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.minor'))) STORED,
    `ver_patch`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.patch'))) STORED,
    `ver_build`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.build'))) STORED,
    `ver_rev`     varchar(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.version.rev'))) STORED,
    `version`     varchar(25) GENERATED ALWAYS AS (concat('v', `ver_major`, '.', `ver_minor`, '.', `ver_patch`, '.',
                                                          `ver_build`, if(((`ver_rev` = '') or (`ver_rev` = NULL)), '',
                                                                          concat('-', `ver_rev`)))) STORED,
    `files`       json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.files'))) VIRTUAL,
    `chksums`     json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.chksums'))) VIRTUAL,
    `upload_date` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `version` (`version`),
    KEY        `fkb_tenant_id` (`tenant_id`),
    CONSTRAINT `fkb_tenant_id` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `modules`
(
    `id`                    int(10) unsigned NOT NULL AUTO_INCREMENT,
    `tenant_id`             int(10) unsigned NOT NULL,
    `service_type`          enum('vxmonitor') NOT NULL,
    `state`                 enum('draft','release') NOT NULL DEFAULT 'draft',
    `config_schema`         json     NOT NULL,
    `default_config`        json     NOT NULL,
    `secure_config_schema`  json     NOT NULL,
    `secure_default_config` json     NOT NULL,
    `static_dependencies`   json     NOT NULL,
    `fields_schema`         json     NOT NULL,
    `action_config_schema`  json     NOT NULL,
    `default_action_config` json     NOT NULL,
    `event_config_schema`   json     NOT NULL,
    `default_event_config`  json     NOT NULL,
    `changelog`             json     NOT NULL,
    `locale`                json     NOT NULL,
    `info`                  json     NOT NULL,
    `name`                  varchar(255) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.name'))) STORED,
    `version`               varchar(50) GENERATED ALWAYS AS (concat(
        json_unquote(json_extract(`info`, _utf8mb4'$.version.major')), '.',
        json_unquote(json_extract(`info`, _utf8mb4'$.version.minor')), '.',
        json_unquote(json_extract(`info`, _utf8mb4'$.version.patch')))) STORED,
    `ver_major`             int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.major'))) STORED,
    `ver_minor`             int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.minor'))) STORED,
    `ver_patch`             int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.patch'))) STORED,
    `tags`                  json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.tags'))) VIRTUAL,
    `fields`                json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.fields'))) VIRTUAL,
    `actions`               json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.actions'))) VIRTUAL,
    `events`                json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.events'))) VIRTUAL,
    `template`              varchar(50) GENERATED ALWAYS AS (json_extract(`info`, _utf8mb4'$.template')) STORED,
    `system`                tinyint(1) GENERATED ALWAYS AS (json_extract(`info`,_utf8mb4 '$.system')) STORED,
    `last_update`           datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `tenant_type_name_version_idx` (`tenant_id`,`service_type`,`name`,`version`),
    KEY        `fkm_tenant_id` (`tenant_id`),
    CONSTRAINT `fkm_tenant_id` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1114 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `privileges`
(
    `id`      int(10) unsigned NOT NULL AUTO_INCREMENT,
    `role_id` int(10) unsigned NOT NULL,
    `name`    varchar(100) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `role_privilege_uniq_key` (`role_id`,`name`),
    KEY        `fkp_role_id` (`role_id`),
    CONSTRAINT `fkp_role_id` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=118 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `services`
(
    `id`             int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`           varchar(32) NOT NULL,
    `tenant_id`      int(10) unsigned NOT NULL,
    `name`           varchar(50) NOT NULL,
    `type`           enum('vxmonitor') NOT NULL,
    `status`         enum('created','active','blocked','removed') NOT NULL,
    `info`           json        NOT NULL,
    `db_name`        varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.db.name'))) STORED,
    `db_user`        varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.db.user'))) STORED,
    `db_pass`        varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.db.pass'))) STORED,
    `db_host`        varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.db.host'))) STORED,
    `db_port`        int(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.db.port'))) STORED,
    `server_host`    varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.server.host'))) STORED,
    `server_port`    int(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.server.port'))) STORED,
    `server_proto`   varchar(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.server.proto'))) STORED,
    `s3_endpoint`    varchar(100) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, '$.s3.endpoint'))) STORED,
    `s3_access_key`  varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, '$.s3.access_key'))) STORED,
    `s3_secret_key`  varchar(50) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, '$.s3.secret_key'))) STORED,
    `s3_bucket_name` varchar(30) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, '$.s3.bucket_name'))) STORED,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`tenant_id`,`name`,`type`),
    UNIQUE KEY `hash_idx` (`hash`),
    KEY        `fks_tenant_id` (`tenant_id`),
    KEY        `services_tenant_id_idx` (`tenant_id`),
    CONSTRAINT `fks_tenant_id` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `users`
(
    `id`        int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`      varchar(32) NOT NULL,
    `type`      enum('local') NOT NULL DEFAULT 'local',
    `mail`      varchar(50) NOT NULL,
    `name`      varchar(70) NOT NULL DEFAULT '',
    `password`  varchar(100)         DEFAULT NULL,
    `status`    enum('created','active','blocked') NOT NULL,
    `role_id`   int(10) unsigned NOT NULL DEFAULT '2',
    `tenant_id` int(10) unsigned NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `mail` (`mail`),
    UNIQUE KEY `hash_idx` (`hash`),
    KEY        `fku_tenant_id` (`tenant_id`),
    KEY        `fku_role_id` (`role_id`),
    CONSTRAINT `fku_role_id` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`),
    CONSTRAINT `fku_tenant_id` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;

SET SESSION sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

INSERT
IGNORE INTO `roles` (`id`, `name`) VALUES
	(0, 'SAdmin'),
	(1, 'Admin'),
    (2, 'User'),
    (100, 'External');

INSERT
IGNORE INTO `tenants` (`id`, `hash`, `status`, `description`) VALUES
	(0, MD5('system'), 'blocked', 'System tenant');

ALTER TABLE `roles` AUTO_INCREMENT = 100;

INSERT
IGNORE INTO `privileges` (`role_id`, `name`) VALUES
    (0, "vxapi.agents.api.create"),
    (0, "vxapi.agents.api.delete"),
    (0, "vxapi.agents.api.edit"),
    (0, "vxapi.agents.api.view"),
    (0, "vxapi.agents.downloads"),
    (0, "vxapi.groups.api.create"),
    (0, "vxapi.groups.api.delete"),
    (0, "vxapi.groups.api.edit"),
    (0, "vxapi.groups.api.view"),
    (0, "vxapi.modules.events"),
    (0, "vxapi.modules.interactive"),
    (0, "vxapi.modules.api.create"),
    (0, "vxapi.modules.api.delete"),
    (0, "vxapi.modules.api.edit"),
    (0, "vxapi.modules.api.view"),
    (0, "vxapi.modules.control.export"),
    (0, "vxapi.modules.control.import"),
    (0, "vxapi.policies.api.create"),
    (0, "vxapi.policies.api.delete"),
    (0, "vxapi.policies.api.edit"),
    (0, "vxapi.policies.api.view"),
    (0, "vxapi.policies.control.link"),
    (0, "vxapi.roles.api.view"),
    (0, "vxapi.services.api.create"),
    (0, "vxapi.services.api.delete"),
    (0, "vxapi.services.api.edit"),
    (0, "vxapi.services.api.view"),
    (0, "vxapi.tenants.api.create"),
    (0, "vxapi.tenants.api.delete"),
    (0, "vxapi.tenants.api.edit"),
    (0, "vxapi.tenants.api.view"),
    (0, "vxapi.users.api.create"),
    (0, "vxapi.users.api.delete"),
    (0, "vxapi.users.api.edit"),
    (0, "vxapi.users.api.view"),
    (0, "vxapi.system.control.update"),
    (0, "vxapi.system.logging.control"),
    (0, "vxapi.system.monitoring.control"),
    (0, "vxapi.templates.create"),
    (0, "vxapi.templates.delete"),
    (0, "vxapi.templates.view"),
    (0, "vxapi.modules.secure-config.view"),
    (0, "vxapi.modules.secure-config.edit"),
    (1, "vxapi.agents.api.create"),
    (1, "vxapi.agents.api.delete"),
    (1, "vxapi.agents.api.edit"),
    (1, "vxapi.agents.api.view"),
    (1, "vxapi.agents.downloads"),
    (1, "vxapi.groups.api.create"),
    (1, "vxapi.groups.api.delete"),
    (1, "vxapi.groups.api.edit"),
    (1, "vxapi.groups.api.view"),
    (1, "vxapi.modules.events"),
    (1, "vxapi.modules.interactive"),
    (1, "vxapi.modules.api.create"),
    (1, "vxapi.modules.api.delete"),
    (1, "vxapi.modules.api.edit"),
    (1, "vxapi.modules.api.view"),
    (1, "vxapi.modules.control.export"),
    (1, "vxapi.modules.control.import"),
    (1, "vxapi.policies.api.create"),
    (1, "vxapi.policies.api.delete"),
    (1, "vxapi.policies.api.edit"),
    (1, "vxapi.policies.api.view"),
    (1, "vxapi.policies.control.link"),
    (1, "vxapi.roles.api.view"),
    (1, "vxapi.services.api.create"),
    (1, "vxapi.services.api.delete"),
    (1, "vxapi.services.api.edit"),
    (1, "vxapi.services.api.view"),
    (1, "vxapi.users.api.create"),
    (1, "vxapi.users.api.delete"),
    (1, "vxapi.users.api.edit"),
    (1, "vxapi.users.api.view"),
    (1, "vxapi.system.control.update"),
    (1, "vxapi.system.logging.control"),
    (1, "vxapi.system.monitoring.control"),
    (1, "vxapi.templates.create"),
    (1, "vxapi.templates.delete"),
    (1, "vxapi.templates.view"),
    (1, "vxapi.modules.secure-config.view"),
    (1, "vxapi.modules.secure-config.edit"),
    (2, "vxapi.agents.api.delete"),
    (2, "vxapi.agents.api.edit"),
    (2, "vxapi.agents.api.view"),
    (2, "vxapi.groups.api.create"),
    (2, "vxapi.groups.api.delete"),
    (2, "vxapi.groups.api.edit"),
    (2, "vxapi.groups.api.view"),
    (2, "vxapi.modules.api.view"),
    (2, "vxapi.modules.events"),
    (2, "vxapi.modules.interactive"),
    (2, "vxapi.policies.api.create"),
    (2, "vxapi.policies.api.delete"),
    (2, "vxapi.policies.api.edit"),
    (2, "vxapi.policies.api.view"),
    (2, "vxapi.policies.control.link"),
    (2, "vxapi.services.api.view"),
    (2, "vxapi.system.monitoring.control"),
    (2, "vxapi.templates.create"),
    (2, "vxapi.templates.view");

-- +migrate Down

DROP TABLE IF EXISTS `binaries`;
DROP TABLE IF EXISTS `modules`;
DROP TABLE IF EXISTS `privileges`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `roles`;
DROP TABLE IF EXISTS `services`;
DROP TABLE IF EXISTS `tenants`;
