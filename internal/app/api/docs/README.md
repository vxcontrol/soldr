### Install Swagger

```
go get github.com/swaggo/swag@v1.7.8
go install github.com/swaggo/swag/cmd/swag
```

### Generate spec

```
cd cmd/api
swag init -g ../../internal/app/api/server/router.go -o ../../internal/app/api/docs/ --parseDependency --parseInternal --parseDepth 2
```
