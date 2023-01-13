### Install Swagger

```
go get github.com/swaggo/swag@v1.8.7
go install github.com/swaggo/swag/cmd/swag
```

### Generate spec

```
cd cmd/api
swag init -g ../../pkg/app/api/server/router.go -o ../../pkg/app/api/docs/ --parseDependency --parseInternal --parseDepth 2
```
