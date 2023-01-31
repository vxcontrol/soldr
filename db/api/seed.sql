INSERT
IGNORE INTO tenants (
    `id`,
    `hash`,
    `status`,
    `description`
 ) VALUES (1, MD5(RAND()), 'active', 'First user tenant');

INSERT
IGNORE INTO services (
    `id`,
    `hash`,
    `tenant_id`,
    `name`,
    `type`,
    `status`,
    `info`
) VALUES (
    1,
    MD5('localhost'),
    1,
    'localhost',
    'vxmonitor',
    'active',
    '{
      "db": {
        "host": "127.0.0.1",
        "name": "vx_instance",
        "pass": "password",
        "port": 3306,
        "user": "vxcontrol"
      },
      "s3": {
        "access_key": "accesskey",
        "secret_key": "secretkey",
        "bucket_name": "vxinstance",
        "endpoint": "http://127.0.0.1:9000"
      },
      "server": {
        "host": "127.0.0.1",
        "port": 8443,
        "proto": "wss"
      }
    }'
);

INSERT
IGNORE INTO users (
    `id`,
    `hash`,
    `status`,
    `role_id`,
    `tenant_id`,
    `mail`,
    `name`,
    `password`
) VALUES (
    1,
    MD5(RAND()),
    'active',
    1,
    1,
    'admin@vxcontrol.com',
    'admin',
    '$2a$10$deVOk0o1nYRHpaVXjIcyCuRmaHvtoMN/2RUT7w5XbZTeiWKEbXx9q'
);
