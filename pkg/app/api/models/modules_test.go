package models

import (
	"crypto/rand"
	"testing"

	"soldr/pkg/crypto"

	"github.com/stretchr/testify/require"
)

func TestDecryptModuleSecureConfigParams(t *testing.T) {
	encryptor, err := crypto.NewAESEncryptor(func() ([]byte, error) {
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
	require.NoError(t, err)

	prefix := "foo"
	dbencryptor := crypto.NewDBConfigEncryptor(encryptor, prefix)

	moduleParam := make(map[string]ModuleSecureParameter)

	const paramName1, paramName2, paramName3 = "key_1", "key_2", "key_3"
	str, digit, list := "test-test", 100500, []string{"val1", "val2"}

	moduleParam[paramName1] = ModuleSecureParameter{Value: str}
	moduleParam[paramName2] = ModuleSecureParameter{Value: digit}
	moduleParam[paramName3] = ModuleSecureParameter{Value: list}

	module := ModuleS{
		SecureDefaultConfig: moduleParam,
	}

	err = module.EncryptSecureParameters(dbencryptor)
	require.NoError(t, err)
	require.Len(t, module.SecureDefaultConfig, 3)

	require.True(t, module.IsEncrypted(dbencryptor))

	err = module.DecryptSecureParameters(dbencryptor)
	require.NoError(t, err)
	require.Len(t, module.SecureDefaultConfig, 3)

	require.False(t, module.IsEncrypted(dbencryptor))

	require.EqualValues(t, str, module.SecureDefaultConfig[paramName1].Value)
	require.EqualValues(t, digit, module.SecureDefaultConfig[paramName2].Value)
	require.Len(t, module.SecureDefaultConfig[paramName3].Value, 2)
	require.Equal(t, list[0], module.SecureDefaultConfig[paramName3].Value.([]interface{})[0])
	require.Equal(t, list[1], module.SecureDefaultConfig[paramName3].Value.([]interface{})[1])
}
