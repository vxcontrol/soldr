package private

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/api/models"
	srvcontext "soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/utils"
)

type services struct {
	Services []models.Service `json:"services"`
	Total    uint64           `json:"total"`
}

var servicesSQLMappers = map[string]interface{}{
	"id":             "`{{table}}`.id",
	"tenant_id":      "`{{table}}`.tenant_id",
	"hash":           "`{{table}}`.hash",
	"name":           "`{{table}}`.name",
	"type":           "`{{table}}`.type",
	"status":         "`{{table}}`.status",
	"info":           "`{{table}}`.info",
	"db_name":        "`{{table}}`.db_name",
	"db_host":        "`{{table}}`.db_host",
	"db_port":        "`{{table}}`.db_port",
	"server_host":    "`{{table}}`.server_host",
	"server_port":    "`{{table}}`.server_port",
	"server_proto":   "`{{table}}`.server_proto",
	"s3_endpoint":    "`{{table}}`.s3_endpoint",
	"s3_bucket_name": "`{{table}}`.s3_bucket_name",
	"data": "CONCAT(`{{table}}`.name, ' | ', " +
		"`{{table}}`.hash, ' | ', " +
		"`{{table}}`.type, ' | ', " +
		"`{{table}}`.status)",
}

type ServicesService struct {
	db *gorm.DB
}

func NewServicesService(db *gorm.DB) *ServicesService {
	return &ServicesService{
		db: db,
	}
}

// GetServices is a function to return services list view on dashboard
// @Summary Retrieve services list by filters
// @Tags Services
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=services} "services list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting services not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting services"
// @Router /services/ [get]
func (s *ServicesService) GetServices(c *gin.Context) {
	var (
		err   error
		query utils.TableQuery
		resp  services
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		logrus.WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrServicesInvalidRequest, err)
		return
	}

	query.Init("services", servicesSQLMappers)

	rid, _ := srvcontext.GetUint64(c, "rid")
	tid, _ := srvcontext.GetUint64(c, "tid")

	switch rid {
	case models.RoleSAdmin:
	case models.RoleUser, models.RoleAdmin, models.RoleExternal:
		query.SetFilters([]func(db *gorm.DB) *gorm.DB{
			func(db *gorm.DB) *gorm.DB {
				return db.Where("tenant_id = ?", tid)
			},
		})
	default:
		logrus.WithError(nil).Errorf("error filtering user role services: unexpected role")
		response.Error(c, response.ErrInternal, nil)
		return
	}

	if resp.Total, err = query.Query(s.db, &resp.Services); err != nil {
		logrus.WithError(err).Errorf("error finding services")
		response.Error(c, response.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Services); i++ {
		if err = resp.Services[i].Valid(); err != nil {
			logrus.WithError(err).Errorf("error validating service data '%s'", resp.Services[i].Hash)
			response.Error(c, response.ErrServicesInvalidData, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// GetService is a function to return service by hash
// @Summary Retrieve service by hash
// @Tags Services
// @Produce json
// @Param hash path string true "hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp{data=models.Service} "service received successful"
// @Failure 403 {object} utils.errorResp "getting service not permitted
// @Failure 404 {object} utils.errorResp "service not found"
// @Failure 500 {object} utils.errorResp "internal error on getting service"
// @Router /services/{hash} [get]
func (s *ServicesService) GetService(c *gin.Context) {
	var (
		err  error
		hash string = c.Param("hash")
		resp models.Service
	)

	rid, _ := srvcontext.GetUint64(c, "rid")
	tid, _ := srvcontext.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		switch rid {
		case models.RoleSAdmin:
			return db.Where("hash = ?", hash)
		case models.RoleUser, models.RoleAdmin, models.RoleExternal:
			return db.Where("tenant_id = ? AND hash = ?", tid, hash)
		default:
			db.AddError(errors.New("unexpected user role"))
			return db
		}
	}

	if err = s.db.Scopes(scope).Take(&resp).Error; err != nil {
		logrus.WithError(err).Errorf("error finding service by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrServicesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = resp.Valid(); err != nil {
		logrus.WithError(err).Errorf("error validating service data '%s'", resp.Hash)
		response.Error(c, response.ErrServicesInvalidData, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

// CreateService is a function to create new service
// @Summary Create new service
// @Tags Services
// @Accept json
// @Produce json
// @Param json body models.Service true "service model to create from"
// @Success 201 {object} utils.successResp{data=models.Service} "service created successful"
// @Failure 400 {object} utils.errorResp "invalid service request data"
// @Failure 403 {object} utils.errorResp "creating service not permitted"
// @Failure 500 {object} utils.errorResp "internal error on creating service"
// @Router /services/ [post]
func (s *ServicesService) CreateService(c *gin.Context) {
	var (
		err     error
		service models.Service
	)

	if err = c.ShouldBindJSON(&service); err != nil {
		logrus.WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrServicesInvalidRequest, err)
		return
	}

	rid, _ := srvcontext.GetUint64(c, "rid")
	tid, _ := srvcontext.GetUint64(c, "tid")

	switch rid {
	case models.RoleSAdmin:
	case models.RoleUser, models.RoleAdmin, models.RoleExternal:
		service.TenantID = tid
	default:
		logrus.WithError(nil).Errorf("error filtering user role services: unexpected role")
		response.Error(c, response.ErrInternal, nil)
		return
	}

	service.Hash = utils.MakeServiceHash(service.Name)

	if err = s.db.Create(&service).Error; err != nil {
		logrus.WithError(err).Errorf("error creating service")
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusCreated, service)
}

// PatchService is a function to update service by hash
// @Summary Update service
// @Tags Services
// @Produce json
// @Param json body models.Service true "service model to update"
// @Param hash path string true "service hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp{data=models.Service} "service updated successful"
// @Failure 400 {object} utils.errorResp "invalid service request data"
// @Failure 403 {object} utils.errorResp "updating service not permitted"
// @Failure 404 {object} utils.errorResp "service not found"
// @Failure 500 {object} utils.errorResp "internal error on updating service"
// @Router /services/{hash} [put]
func (s *ServicesService) PatchService(c *gin.Context) {
	var (
		err     error
		hash    string = c.Param("hash")
		service models.Service
	)

	if err = c.ShouldBindJSON(&service); err != nil {
		logrus.WithError(err).Errorf("error binding JSON")
		response.Error(c, response.ErrServicesInvalidRequest, err)
		return
	} else if hash != service.Hash {
		logrus.WithError(nil).Errorf("mismatch service hash to requested one")
		response.Error(c, response.ErrServicesInvalidRequest, nil)
		return
	} else if err = service.Valid(); err != nil {
		logrus.WithError(err).Errorf("error validating service JSON")
		response.Error(c, response.ErrServicesInvalidRequest, err)
		return
	}

	rid, _ := srvcontext.GetUint64(c, "rid")
	tid, _ := srvcontext.GetUint64(c, "tid")
	if rid == models.RoleExternal {
		logrus.WithError(nil).Errorf("error: no rights to patch service")
		response.Error(c, response.ErrNotPermitted, nil)
		return
	}
	scope := func(db *gorm.DB) *gorm.DB {
		switch rid {
		case models.RoleSAdmin:
			return db.Where("hash = ?", hash)
		case models.RoleUser, models.RoleAdmin, models.RoleExternal:
			return db.Where("tenant_id = ? AND hash = ?", tid, hash)
		default:
			db.AddError(errors.New("unexpected user role"))
			return db
		}
	}

	public_info := []interface{}{"info", "name", "status"}
	err = s.db.Scopes(scope).Select("", public_info...).Save(&service).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithError(nil).Errorf("error updating service by hash '%s', service not found", hash)
		response.Error(c, response.ErrServicesNotFound, err)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("error updating service by hash '%s'", hash)
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, service)
}

// DeleteService is a function to delete service by hash
// @Summary Delete service by hash
// @Tags Services
// @Produce json
// @Param hash path string true "hash in hex format (md5)" minlength(32) maxlength(32)
// @Success 200 {object} utils.successResp "service deleted successful"
// @Failure 403 {object} utils.errorResp "deleting service not permitted"
// @Failure 404 {object} utils.errorResp "service not found"
// @Failure 500 {object} utils.errorResp "internal error on deleting service"
// @Router /services/{hash} [delete]
func (s *ServicesService) DeleteService(c *gin.Context) {
	var (
		err     error
		hash    string = c.Param("hash")
		service models.Service
	)

	rid, _ := srvcontext.GetUint64(c, "rid")
	tid, _ := srvcontext.GetUint64(c, "tid")
	if rid == models.RoleExternal {
		logrus.WithError(nil).Errorf("error: no rights to delete service")
		response.Error(c, response.ErrNotPermitted, nil)
		return
	}
	scope := func(db *gorm.DB) *gorm.DB {
		switch rid {
		case models.RoleSAdmin:
			return db.Where("hash = ?", hash)
		case models.RoleUser, models.RoleAdmin, models.RoleExternal:
			return db.Where("tenant_id = ? AND hash = ?", tid, hash)
		default:
			db.AddError(errors.New("unexpected user role"))
			return db
		}
	}

	if err = s.db.Scopes(scope).Take(&service).Error; err != nil {
		logrus.WithError(err).Errorf("error finding service by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrServicesNotFound, err)
		} else {
			response.Error(c, response.ErrInternal, err)
		}
		return
	} else if err = service.Valid(); err != nil {
		logrus.WithError(err).Errorf("error validating service data '%s'", service.Hash)
		response.Error(c, response.ErrServicesInvalidData, err)
		return
	}

	if err = s.db.Delete(&service).Error; err != nil {
		logrus.WithError(err).Errorf("error deleting service by hash '%s'", hash)
		response.Error(c, response.ErrInternal, err)
		return
	}

	response.Success(c, http.StatusOK, struct{}{})
}
