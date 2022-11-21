package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
)

func main() {
	key, err := GenerateKey()
	if err != nil {
		panic(err)
	}

	const path = "internal/app/api/utils/dbencryptor/sec-store-key.txt"
	err = ioutil.WriteFile(path, []byte(key), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("key saved successfully: %s\n", path)
}

func GenerateKey() (string, error) {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
