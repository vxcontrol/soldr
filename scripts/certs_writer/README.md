# Certificates writer

This is a utility that allows to generate a set of certificates for the VX server and agents

To generate everything at once run:

```bash
./certs_writer -cert_gen=true --dst=./certs --server_name=example
```
or
```bash
make generate-all
```
