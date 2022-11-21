generate a key which is used to encrypt/decrypt secure module configs json data:  

```
go run keygen.go
```
The key then is used when building the app using env variable `DB_ENCRYPT_KEY`.

The key value must be used in vxserver repo to decrypt secured params.  

DO NOT OVERWRITE THE FILE to allow backward compatibility.
