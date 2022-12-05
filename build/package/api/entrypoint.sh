#!/bin/bash

export API_SSL_KEY=ssl/server.key
export API_SSL_CRT=ssl/server.crt
API_SSL_CSR=ssl/server.csr
API_SSL_CA_KEY=ssl/server_ca.key
API_SSL_CA_CRT=ssl/server_ca.crt

if [[ -f "$API_SSL_KEY" && -f "$API_SSL_CRT" ]]; then
    echo "API server ssl crt and key already exist."
elif [ "$API_USE_SSL" = "true" ]; then
    echo "Gen API server ssl key and crt."
    openssl genrsa -out ${API_SSL_CA_KEY} 4096
    openssl req \
        -new -x509 -days 3650 \
        -key ${API_SSL_CA_KEY} \
        -subj "/C=RU/ST=MO/L=MO/O=VXControl/OU=Org/CN=VXControl SOLDR Root CA" \
        -out ${API_SSL_CA_CRT}
    openssl req \
        -newkey rsa:4096 \
        -sha256 \
        -nodes \
        -keyout ${API_SSL_KEY} \
        -subj "/C=RU/ST=MO/L=MO/O=VXControl/OU=Org/CN=vxapi.local" \
        -out ${API_SSL_CSR}
    openssl x509 -req \
        -days 730 \
        -extfile <(printf "subjectAltName=DNS:vxapi.local\nkeyUsage=critical,digitalSignature,keyAgreement\nextendedKeyUsage=serverAuth") \
        -in ${API_SSL_CSR} \
        -CA ${API_SSL_CA_CRT} -CAkey ${API_SSL_CA_KEY} -CAcreateserial \
        -out ${API_SSL_CRT}
    cat ${API_SSL_CA_CRT} >> ${API_SSL_CRT}

    chmod g+r ${API_SSL_KEY}
    chmod g+r ${API_SSL_CA_KEY}
fi

$@
