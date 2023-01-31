package private

import (
	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/crypto"

	"github.com/gin-gonic/gin"
)

func getService(c *gin.Context) *models.Service {
	var sv *models.Service

	if val, ok := c.Get("SV"); !ok {
		logger.FromContext(c).Errorf("error getting vxservice instance from context")
	} else if sv = val.(*models.Service); sv == nil {
		logger.FromContext(c).Errorf("got nil value vxservice instance from context")
	}

	return sv
}

func getDBEncryptor(c *gin.Context) crypto.IDBConfigEncryptor {
	var encryptor crypto.IDBConfigEncryptor

	if cr, ok := c.Get("crp"); !ok {
		logger.FromContext(c).Errorf("error getting secure config encryptor from context")
	} else if encryptor = cr.(crypto.IDBConfigEncryptor); encryptor == nil {
		logger.FromContext(c).Errorf("got nil value secure config encryptor from context")
	}

	return encryptor
}
