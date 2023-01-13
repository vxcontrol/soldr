package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecrypt(t *testing.T) {
	data := "message"
	key := []byte("my_key")
	ct, err := Encrypt([]byte(data), key)
	require.NoError(t, err)
	require.NotEmpty(t, ct)

	result, err := Decrypt(ct, key)
	require.NoError(t, err)
	require.Equal(t, data, string(result))
}
