FROM minio/mc as minio_client

FROM alpine

COPY --from=minio_client /usr/bin/mc /usr/bin/mc

RUN mkdir -p /opt/vxdbmigrate/utils && \
    apk add mysql-client bash

# Entrypoint script
COPY build/package/dbmigrate/entrypoint.sh /opt/vxdbmigrate/
COPY build/package/api/utils /opt/vxdbmigrate/utils

WORKDIR /opt/vxdbmigrate

RUN chmod +x /opt/vxdbmigrate/entrypoint.sh

ENTRYPOINT ["/opt/vxdbmigrate/entrypoint.sh"]
