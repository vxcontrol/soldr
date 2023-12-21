# SBH generator

Generates an SBH token (see the description of the communication protocol between the SOLDR server and agents).

## Usage

Example of usage:

```go
./sbh_generator --key=../../security/certs/example/vxca.key --expires=2026-01-01T12:00:00+00:00 --file=../../security/vconf/lic/sbh.json --version=example --force=true
```
