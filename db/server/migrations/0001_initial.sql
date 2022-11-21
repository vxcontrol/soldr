-- +migrate Up

CREATE TABLE IF NOT EXISTS `agents`
(
    `id`             int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`           varchar(32)  NOT NULL,
    `group_id`       int(10) unsigned NOT NULL,
    `ip`             varchar(50)  NOT NULL,
    `description`    varchar(255) NOT NULL,
    `version`        varchar(20)  NOT NULL,
    `info`           json         NOT NULL,
    `status`         enum('connected','disconnected') NOT NULL,
    `auth_status`    enum('authorized','unauthorized','blocked') NOT NULL,
    `os_type`        varchar(30) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.os.type'))) STORED,
    `os_arch`        varchar(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.os.arch'))) STORED,
    `os_name`        varchar(30) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.os.name'))) STORED,
    `hostname`       varchar(70) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.net.hostname'))) STORED,
    `connected_date` datetime              DEFAULT NULL,
    `created_date`   datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`     datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `deleted_at`     datetime              DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `hash_deleted_idx` (`hash`,`deleted_at`),
    KEY              `agents_deleted_idx` (`deleted_at`),
    KEY              `group_id_idx` (`group_id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `binaries`
(
    `id`        int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`      varchar(32) NOT NULL,
    `info`      json        NOT NULL,
    `ver_major` int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.major'))) STORED,
    `ver_minor` int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.minor'))) STORED,
    `ver_patch` int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.patch'))) STORED,
    `ver_build` int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.build'))) STORED,
    `ver_rev`   varchar(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.version.rev'))) STORED,
    `version`   varchar(25) GENERATED ALWAYS AS (concat('v', `ver_major`, '.', `ver_minor`, '.', `ver_patch`, '.',
                                                        `ver_build`, if(((`ver_rev` = '') or (`ver_rev` = NULL)), '',
                                                                        concat('-', `ver_rev`)))) STORED,
    `files`     json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.files'))) VIRTUAL,
    `chksums`   json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.chksums'))) VIRTUAL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `version` (`version`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `events`
(
    `id`        int(10) unsigned NOT NULL AUTO_INCREMENT,
    `module_id` int(10) unsigned NOT NULL,
    `agent_id`  int(10) unsigned NOT NULL,
    `info`      json     NOT NULL,
    `name`      varchar(100) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.name'))) STORED,
    `data`      json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.data'))) VIRTUAL,
    `data_text` text GENERATED ALWAYS AS (json_unquote(json_extract(`info`, '$.data'))) STORED,
    `actions`   json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.actions'))) STORED,
    `uniq`      varchar(255) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.uniq'))) STORED,
    `date`      datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_event_idx` (`module_id`,`agent_id`,`uniq`),
    KEY         `module_and_agent` (`module_id`,`agent_id`),
    KEY         `module_id` (`module_id`),
    KEY         `agent_id` (`agent_id`),
    KEY         `name` (`name`),
    KEY         `uniq` (`uniq`),
    FULLTEXT KEY `data_text` (`data_text`)
) ENGINE=InnoDB AUTO_INCREMENT=1384 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `external_connections`
(
    `id`          int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`        varchar(32)  NOT NULL,
    `description` varchar(255) NOT NULL,
    `type`        enum('browser','external') NOT NULL,
    `info`        json         NOT NULL,
    `ver_major`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.major'))) STORED,
    `ver_minor`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.minor'))) STORED,
    `ver_patch`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.patch'))) STORED,
    `ver_build`   int(10) unsigned GENERATED ALWAYS AS (json_unquote(json_extract(`info`,_utf8mb4 '$.version.build'))) STORED,
    `ver_rev`     varchar(10) GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.version.rev'))) STORED,
    `version`     varchar(25) GENERATED ALWAYS AS (concat('v', `ver_major`, '.', `ver_minor`, '.', `ver_patch`, '.',
                                                          `ver_build`, if(((`ver_rev` = '') or (`ver_rev` = NULL)), '',
                                                                          concat('-', `ver_rev`)))) STORED,
    `chksums`     json GENERATED ALWAYS AS (json_unquote(json_extract(`info`, _utf8mb4'$.chksums'))) VIRTUAL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `hash` (`hash`),
    UNIQUE KEY `type_version` (`type`,`version`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `groups`
(
    `id`           int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`         varchar(32) NOT NULL,
    `info`         json        NOT NULL,
    `created_date` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `deleted_at`   datetime             DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `hash_deleted_idx` (`hash`,`deleted_at`),
    KEY            `groups_deleted_idx` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1005 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `groups_to_policies`
(
    `id`        int(10) unsigned NOT NULL AUTO_INCREMENT,
    `group_id`  int(10) unsigned NOT NULL,
    `policy_id` int(10) unsigned NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=20 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `modules`
(
    `id`                    int(10) unsigned NOT NULL AUTO_INCREMENT,
    `policy_id`             int(10) unsigned NOT NULL,
    `status`                enum('joined','inactive') NOT NULL,
    `state`                 enum('draft','release') NOT NULL,
    `join_date`             datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `config_schema`         json     NOT NULL,
    `default_config`        json     NOT NULL,
    `secure_config_schema`  json     NOT NULL,
    `secure_default_config` json     NOT NULL,
    `secure_current_config` json     NOT NULL,
    `static_dependencies`   json     NOT NULL,
    `dynamic_dependencies`  json     NOT NULL,
    `current_config`        json     NOT NULL,
    `fields_schema`         json     NOT NULL,
    `action_config_schema`  json     NOT NULL,
    `default_action_config` json     NOT NULL,
    `current_action_config` json     NOT NULL,
    `event_config_schema`   json              DEFAULT NULL,
    `default_event_config`  json              DEFAULT NULL,
    `current_event_config`  json              DEFAULT NULL,
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
    `dependencies`          json GENERATED ALWAYS AS (json_unquote(json_extract(
        json_merge_preserve(`static_dependencies`, `dynamic_dependencies`), _utf8mb4'$[*].module_name'))) VIRTUAL,
    `template`              varchar(50) GENERATED ALWAYS AS (json_extract(`info`, _utf8mb4'$.template')) STORED,
    `system`                tinyint(1) GENERATED ALWAYS AS (json_extract(`info`,_utf8mb4 '$.system')) STORED,
    `last_module_update`    datetime NOT NULL,
    `last_update`           datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`            datetime          DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `policy_id_module_name_deleted_idx` (`policy_id`,`name`,`deleted_at`),
    KEY                     `name` (`name`),
    KEY                     `modules_deleted_idx` (`deleted_at`),
    KEY                     `status_idx` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=1027 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `policies`
(
    `id`           int(10) unsigned NOT NULL AUTO_INCREMENT,
    `hash`         varchar(32) NOT NULL,
    `info`         json        NOT NULL,
    `created_date` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `deleted_at`   datetime             DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `hash_deleted_idx` (`hash`,`deleted_at`),
    KEY            `policies_deleted_idx` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1009 DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `upgrade_tasks`
(
    `id`           int(10) unsigned NOT NULL AUTO_INCREMENT,
    `batch`        varchar(32) NOT NULL,
    `agent_id`     int(10) unsigned NOT NULL,
    `version`      varchar(20) NOT NULL,
    `created`      datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_upgrade` datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `status`       enum('new','running','ready','failed') NOT NULL DEFAULT 'new',
    `reason`       varchar(150)         DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY            `agent_id` (`agent_id`),
    KEY            `status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- +migrate Down

DROP TABLE IF EXISTS `agents`;
DROP TABLE IF EXISTS `binaries`;
DROP TABLE IF EXISTS `events`;
DROP TABLE IF EXISTS `external_connections`;
DROP TABLE IF EXISTS `groups`;
DROP TABLE IF EXISTS `groups_to_policies`;
DROP TABLE IF EXISTS `modules`;
DROP TABLE IF EXISTS `policies`;
DROP TABLE IF EXISTS `upgrade_tasks`;
