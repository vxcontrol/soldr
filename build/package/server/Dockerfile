FROM debian:buster-slim

RUN mkdir -p /opt/vxserver/{bin,data,logs}

COPY preparing.sh 	                    /opt/vxserver/bin/
COPY ../../artifacts/server/version 	/opt/vxserver/bin/
COPY ../../bin/server 			        /opt/vxserver/bin/
COPY ../../../db/server/migrations 	    /opt/vxserver/migrations
COPY build/package/server/vconf 	    /opt/vxserver/vconf
COPY build/package/server/crts/server 	/opt/vxserver/crts/server

WORKDIR /opt/vxserver

RUN chmod +x /opt/vxserver/bin/preparing.sh && \
    /opt/vxserver/bin/preparing.sh

ENTRYPOINT ["/opt/vxserver/bin/server"]
