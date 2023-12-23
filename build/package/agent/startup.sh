#!/bin/bash

export LANG=C.UTF-8

export VERSION=$(cat /opt/vxbinaries/version)
export SEMVER_REGEX="^(v)?(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\.(0|[1-9][0-9]*))?(\\.(0|[1-9][0-9]*))?(\\-([0-9A-Za-z-]*))?$"

function parse_sem_ver {
    if [[ "$VERSION" =~ $SEMVER_REGEX ]]; then
        local matches=("${BASH_REMATCH[@]:0}")
        if [ "${matches[1]}" = "v" ]; then
            matches=("${matches[@]:1}")
        fi

        local major=${matches[1]}
        local minor=${matches[2]}
        local patch=0
        local build=0
        local revision=

        if [ -n "${matches[4]}" ]; then
            patch=${matches[4]}
        fi
        if [ -n "${matches[6]}" ]; then
            build=${matches[6]}
        fi
        if [ -n "${matches[8]}" ]; then
            revision=${matches[8]}
        fi

        VERSION="v${major}.${minor}.${patch}.${build}"
        if [ -n "${revision}" ]; then
            VERSION="${VERSION}-${revision}"
        fi
        echo "{\"major\": $major, \"minor\": $minor, \"patch\": $patch, \"build\": $build, \"rev\": \"$revision\"}"
    else
        echo "version $VERSION does not match the semver scheme '(v)X.Y.Z(+BUILD)(+REVISION)'" >&2
        echo "{}"
    fi
}

function update_info {
    CHTYPE=$2
    IFS=$'\n' sumres=($1)
    for line in "${sumres[@]}"
    do
        IFS=' ' read -r -a alfds <<< "$line"
        FPATH="vxagent/$VERSION/${alfds[1]}"
        FCSUM="${alfds[0]}"
        FILE="{\"files\": [\"${FPATH}\"], \"chksums\": {\"${FPATH}\": {\"${CHTYPE}\": \"${FCSUM}\"}}}"
        INFO=$(jq -c -n --argjson info $INFO --argjson file $FILE '$info * $file * {files: [$info.files + $file.files][0] | unique}')
    done
}

# Normalize input semantic version to own format
INFO="{\"version\": $(parse_sem_ver)}"

# Enrich agent binary record to check sums information about input files
update_info "$(md5sum */*/*)" "md5"
update_info "$(sha256sum */*/*)" "sha256"

# Debug print result of agent binary info
echo $INFO | jq .

# Test connection to minio
while true; do
    mc config host add vxm "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" 2>&1 1>/dev/null
    if [ $? -eq 0 ]; then
        echo "connect to minio was successful"
        break
    fi
    echo "failed to connect to minio"
    sleep 1
done

mc mb --ignore-existing vxm/$MINIO_BUCKET_NAME

# Check and upload binary file if it isn't exist
for file in $(ls -d */*/*); do
    lfile=/opt/vxbinaries/${file}
    rfile=vxm/$MINIO_BUCKET_NAME/vxagent/${VERSION}/${file}
    if [ -z $(mc ls ${rfile}) ]; then
        mc cp ${lfile} ${rfile}
    else
        echo "binary file '${rfile}' already exists"
    fi
done

# Test connection to mysql
while true; do
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -e ";" 2>&1 1>/dev/null
    if [ $? -eq 0 ]; then
        echo "connect to mysql was successful"
        break
    fi
    echo "failed to connect to mysql"
    sleep 1
done

# Waitign migrations from vxui into mysql
GET_MIGRATION="SELECT count(*) FROM gorp_migrations WHERE id = '0001_initial.sql';"
while true; do
    MIGRATION=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -Nse "$GET_MIGRATION" 2>/dev/null)
    if [[ $? -eq 0 && $MIGRATION -eq 1 ]]; then
        echo "vxapi migrations was found"
        break
    fi
    echo "failed to update binaries table"
    sleep 1
done

# Prepare SQL query to insert new binaries version
cat <<EOT > /opt/vxbinaries/upload_binaries.sql
SET @hash = MD5('${VERSION}'),
    @info = '${INFO}';
INSERT INTO \`binaries\`
    (\`hash\`, \`tenant_id\`, \`type\`, \`info\`)
VALUES
    (@hash, 0, 'vxagent', @info)
ON DUPLICATE KEY UPDATE
    hash = @hash;
EOT

mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" < /opt/vxbinaries/upload_binaries.sql

echo "done"

sleep infinity
