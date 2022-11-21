FROM debian:buster-slim

RUN mkdir -p /opt/api/{logs,ssl}

COPY build/bin/api                   /opt/api/bin/
COPY web/dist/                       /opt/api/static
COPY db/api/migrations               /opt/api/migrations
COPY build/package/api/resources     /opt/api/resources
COPY build/package/api/templates     /opt/api/templates
COPY build/package/api/crts          /opt/api/crts
COPY build/package/api/entrypoint.sh /opt/api/bin/

RUN chmod +x /opt/api/bin/entrypoint.sh

WORKDIR /opt/api

ENTRYPOINT ["/opt/api/bin/entrypoint.sh", "/opt/api/bin/api"]
