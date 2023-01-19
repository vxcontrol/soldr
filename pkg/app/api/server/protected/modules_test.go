package protected

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"soldr/pkg/app/api/models"
	"soldr/pkg/crypto"
)

const (
	testKey    = "YvTnpzagroGNKyU3LDxpBG9nZFrA5/pfLU03Yfs6Hmg="
	testPrefix = "test"
)

func TestDecryptModuleSecureConfigParams(t *testing.T) {
	var (
		encryptor   = getTestEncryptor(t)
		moduleParam = make(map[string]models.ModuleSecureParameter)
	)

	const paramName1, paramName2, paramName3 = "key_1", "key_2", "key_3"
	str, digit, list := "test-test", 100500, []string{"val1", "val2"}

	moduleParam[paramName1] = models.ModuleSecureParameter{Value: str}
	moduleParam[paramName2] = models.ModuleSecureParameter{Value: digit}
	moduleParam[paramName3] = models.ModuleSecureParameter{Value: list}

	module := models.ModuleS{
		SecureDefaultConfig: moduleParam,
	}

	err := module.EncryptSecureParameters(encryptor)
	require.NoError(t, err)
	require.Len(t, module.SecureDefaultConfig, 3)

	require.True(t, module.IsEncrypted(encryptor))

	err = module.DecryptSecureParameters(encryptor)
	require.NoError(t, err)
	require.Len(t, module.SecureDefaultConfig, 3)

	require.False(t, module.IsEncrypted(encryptor))

	require.EqualValues(t, str, module.SecureDefaultConfig[paramName1].Value)
	require.EqualValues(t, digit, module.SecureDefaultConfig[paramName2].Value)
	require.Len(t, module.SecureDefaultConfig[paramName3].Value, 2)
	require.Equal(t, list[0], module.SecureDefaultConfig[paramName3].Value.([]interface{})[0])
	require.Equal(t, list[1], module.SecureDefaultConfig[paramName3].Value.([]interface{})[1])
}

func getTestEncryptor(t *testing.T) *crypto.DBConfigEncryptor {
	encryptor, err := crypto.NewAESEncryptor(func() ([]byte, error) {
		b, err := base64.StdEncoding.DecodeString(testKey)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
	require.NoError(t, err)

	return crypto.NewDBConfigEncryptor(encryptor, testPrefix)
}
