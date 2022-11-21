package hardening

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/internal/app/server/certs"
	"soldr/internal/app/server/mmodule/hardening/v1/approver"
	v1Validator "soldr/internal/app/server/mmodule/hardening/v1/validator"
	"soldr/internal/storage"
	"soldr/internal/vxproto"
)

type ConnectionValidatorFactory struct {
	versions map[string]func() (vxproto.ServerConnectionValidator, error)
}

func NewConnectionValidatorFactory(
	ctx context.Context,
	gdbc *gorm.DB,
	fs storage.IFileReader,
	store interface{},
	basePath string,
	certsProvider certs.Provider,
	approver approver.Approver,
	logger *logrus.Entry,
) *ConnectionValidatorFactory {
	return &ConnectionValidatorFactory{
		versions: map[string]func() (vxproto.ServerConnectionValidator, error){
			"v1": func() (vxproto.ServerConnectionValidator, error) {
				return v1Validator.NewConnectionValidator(
					ctx,
					gdbc,
					fs,
					store,
					basePath,
					certsProvider,
					approver,
					logger,
				)
			},
		},
	}
}

func (f *ConnectionValidatorFactory) NewValidator(version string) (vxproto.ServerConnectionValidator, error) {
	fnNewValidator, ok := f.versions[version]
	if !ok {
		return nil, fmt.Errorf("no validator constructor registered for the version %s", version)
	}
	v, err := fnNewValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create a connection validator: %w", err)
	}
	return v, nil
}
