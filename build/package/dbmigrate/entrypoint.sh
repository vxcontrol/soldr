#!/bin/bash

# Test connection to mysql
while true; do
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -e ";" 2>&1 1>/dev/null
    if [ $? -eq 0 ]; then
        echo "connect to mysql was successful"
        break
    fi
    echo "failed to connect to mysql"
    sleep 10
done

echo "Creating user for vxserver"
mysql --host=${DB_HOST} --user=${MYSQL_ROOT_USER} --password=${MYSQL_ROOT_PASSWORD} --port=${DB_PORT} --execute="CREATE DATABASE IF NOT EXISTS ${DB_NAME};"
mysql --host=${DB_HOST} --user=${MYSQL_ROOT_USER} --password=${MYSQL_ROOT_PASSWORD} --port=${DB_PORT} --execute="ALTER DATABASE ${DB_NAME} DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_unicode_ci;"
mysql --host=${DB_HOST} --user=${MYSQL_ROOT_USER} --password=${MYSQL_ROOT_PASSWORD} --port=${DB_PORT} --execute="CREATE DATABASE IF NOT EXISTS ${AGENT_SERVER_DB_NAME};"
mysql --host=${DB_HOST} --user=${MYSQL_ROOT_USER} --password=${MYSQL_ROOT_PASSWORD} --port=${DB_PORT} --execute="ALTER DATABASE ${AGENT_SERVER_DB_NAME} DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_unicode_ci;"
mysql --host=${DB_HOST} --user=${MYSQL_ROOT_USER} --password=${MYSQL_ROOT_PASSWORD} --port=${DB_PORT} --execute="CREATE USER IF NOT EXISTS '${AGENT_SERVER_DB_USER}' IDENTIFIED BY '${AGENT_SERVER_DB_PASS}';"
mysql --host=${DB_HOST} --user=${MYSQL_ROOT_USER} --password=${MYSQL_ROOT_PASSWORD} --port=${DB_PORT} --execute="GRANT ALL PRIVILEGES ON ${AGENT_SERVER_DB_NAME}.* TO ${AGENT_SERVER_DB_USER}@'%';"
echo "done"

mc config host add vxm "${MINIO_ENDPOINT}" ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY} 2>/dev/null
mc mb --ignore-existing vxm/${MINIO_BUCKET_NAME}
mc cp --recursive /opt/vxdbmigrate/utils vxm/${MINIO_BUCKET_NAME}
mc config host add vxinst "${MINIO_ENDPOINT}" ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY} 2>/dev/null
mc mb --ignore-existing vxinst/${AGENT_SERVER_MINIO_BUCKET_NAME}
mc cp --recursive /opt/vxdbmigrate/utils vxinst/${AGENT_SERVER_MINIO_BUCKET_NAME}

cat <<EOT > /opt/vxdbmigrate/seed.sql
INSERT
IGNORE INTO \`tenants\` (
    \`id\`,
    \`hash\`,
    \`status\`,
    \`description\`
 ) VALUES (1, MD5(RAND()), 'active', 'First user tenant');

INSERT
IGNORE INTO \`services\` (
    \`id\`,
    \`hash\`,
    \`tenant_id\`,
    \`name\`,
    \`type\`,
    \`status\`,
    \`info\`
) VALUES (
    1,
    MD5('localhost'),
    1,
    'localhost',
    'vxmonitor',
    'active',
    '{
      "db": {
        "host": "$DB_HOST",
        "name": "$AGENT_SERVER_DB_NAME",
        "pass": "$AGENT_SERVER_DB_PASS",
        "port": $DB_PORT,
        "user": "$AGENT_SERVER_DB_USER"
      },
      "s3": {
        "access_key": "$MINIO_ACCESS_KEY",
        "secret_key": "$MINIO_SECRET_KEY",
        "bucket_name": "$AGENT_SERVER_MINIO_BUCKET_NAME",
        "endpoint": "$MINIO_ENDPOINT"
      },
      "server": {
        "host": "$AGENT_SERVER_HOST",
        "port": 8443,
        "proto": "wss"
      }
    }'
);

INSERT
IGNORE INTO \`users\` (
    \`id\`,
    \`hash\`,
    \`status\`,
    \`role_id\`,
    \`tenant_id\`,
    \`mail\`,
    \`name\`,
    \`password\`,
    \`password_change_required\`
) VALUES (
    1,
    MD5(RAND()),
    'active',
    1,
    1,
    'admin@vxcontrol.com',
    'admin',
    '\$2a\$10\$deVOk0o1nYRHpaVXjIcyCuRmaHvtoMN/2RUT7w5XbZTeiWKEbXx9q',
    'true'
);

EOT

# Waiting migrations from vxui into mysql to upload seed data
GET_MIGRATION="SELECT count(*) FROM gorp_migrations WHERE id = '0002_add_user_password_change_flag.sql';"
while true; do
    MIGRATION=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -Nse "$GET_MIGRATION" 2>/dev/null)
    if [[ $? -eq 0 && $MIGRATION -eq 1 ]]; then
        echo "vxapi migrations was found"
        break
    fi
    echo "wait vxapi service initialization"
    sleep 1
done

mysql --host=${DB_HOST} --user=${DB_USER} --password=${DB_PASS} --port=${DB_PORT} "$DB_NAME" < /opt/vxdbmigrate/seed.sql

sleep infinity
