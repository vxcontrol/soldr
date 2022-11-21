package dbencryptor

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"soldr/internal/crypto"

	"github.com/stretchr/testify/require"
)

func TestDecrypt(t *testing.T) {
	encryptor, err0 := crypto.NewAESEncryptor(generateTestKey)
	if err0 != nil {
		t.Fatalf("%s", err0)
	}

	prefix := "foo"
	dbencryptor := crypto.NewDBConfigEncryptor(encryptor, prefix)

	testCases := []struct {
		name string
		val  interface{}
	}{
		{
			name: "string",
			val:  "Hello world!",
		}, {
			name: "int",
			val:  105000,
		}, {
			name: "struct",
			val: map[string]interface{}{
				"val1": 1.1,
				"val2": []interface{}{"s1", "s2"}},
		}, {
			name: "json",
			val:  "{\"key1\":1.1,\"key2\":[\"s1\",\"s2\"]}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var value = tc.val

			b, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("%s", err)
			}

			result, err := dbencryptor.EncryptValue(b)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(result, prefix+".") || !dbencryptor.IsFormatMatch(result) {
				t.Fatalf("invalid output format")
			}

			decoded, err := base64.StdEncoding.DecodeString(result[len(prefix)+1:])
			if err != nil {
				t.Fatal(err)
			}

			decrypted1, err := encryptor.Decrypt(decoded)
			if err != nil {
				t.Fatal(err)
			}

			decrypted2, err := dbencryptor.DecryptValue(result)
			if err != nil {
				t.Fatal(err)
			}

			if string(decrypted1) != string(decrypted2) {
				t.Fatalf("expected: %s; actual: %s", string(decrypted1), string(decrypted2))
			}

			var actual interface{}
			err = json.Unmarshal(decrypted2, &actual)
			if err != nil {
				t.Fatalf("%s", err)
			}

			require.EqualValues(t, value, actual)
		})
	}
}

func generateTestKey() ([]byte, error) {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
