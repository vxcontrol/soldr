package protected

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/response"
	storage2 "soldr/pkg/app/api/storage"
	useraction "soldr/pkg/app/api/user_action"
	"soldr/pkg/storage"
)

type binaries struct {
	Binaries []models.Binary `json:"binaries"`
	Total    uint64          `json:"total"`
}

var binariesSQLMappers = map[string]interface{}{
	"id":        "`{{table}}`.id",
	"tenant_id": "`{{table}}`.tenant_id",
	"hash":      "`{{table}}`.hash",
	"ver_major": "`{{table}}`.ver_major",
	"ver_minor": "`{{table}}`.ver_minor",
	"ver_patch": "`{{table}}`.ver_patch",
	"ver_build": "`{{table}}`.ver_build",
	"ver_rev":   "`{{table}}`.ver_rev",
	"version":   "`{{table}}`.version",
	"files":     storage2.BinaryFilesMapper,
	"chksum":    storage2.BinaryChksumsMapper,
	"data": "CONCAT(`{{table}}`.version, ' | ', " +
		"`{{table}}`.hash, ' | ', " +
		"`{{table}}`.type, ' | ', " +
		"`{{table}}`.files)",
}

type BinariesService struct {
	db               *gorm.DB
	userActionWriter useraction.Writer
}

func NewBinariesService(
	db *gorm.DB,
	userActionWriter useraction.Writer,
) *BinariesService {
	return &BinariesService{
		db:               db,
		userActionWriter: userActionWriter,
	}
}

func patchOrderingByVersion(query *storage2.TableQuery) {
	if query.Sort.Prop != "version" {
		return
	}

	var arrow string
	switch query.Sort.Order {
	case "ascending":
		arrow = "ASC"
	case "descending":
		arrow = "DESC"
	}
	query.Sort.Order = ""
	query.Sort.Prop = ""
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			if arrow == "" {
				return db
			}
			return db.
				Order(fmt.Sprintf("ver_major %s", arrow)).
				Order(fmt.Sprintf("ver_minor %s", arrow)).
				Order(fmt.Sprintf("ver_patch %s", arrow)).
				Order(fmt.Sprintf("ver_build %s", arrow))
		},
	})
}

// GetAgentBinaries is a function to return agent binaries list
// @Summary Retrieve agent binaries list by filters
// @Tags Binaries
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=binaries} "agent binaries list received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting agent binaries not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting agent binaries"
// @Router /binaries/vxagent [get]
func (s *BinariesService) GetAgentBinaries(c *gin.Context) {
	var (
		err   error
		query storage2.TableQuery
		resp  binaries
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error binding query")
		response.Error(c, response.ErrAgentBinariesInvalidRequest, err)
		return
	}

	tid, _ := context.GetUint64(c, "tid")
	patchOrderingByVersion(&query)

	query.Init("binaries", binariesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.
				Where("tenant_id IN (0, ?)", tid).
				Where("type LIKE ?", "vxagent").
				Where("NOT ISNULL(version)")
		},
	})

	if resp.Total, err = query.Query(s.db, &resp.Binaries); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error finding agent binaries")
		response.Error(c, response.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Binaries); i++ {
		if err = resp.Binaries[i].Valid(); err != nil {
			logger.FromContext(c).WithError(err).Errorf("error validating agent binaries data '%s'", resp.Binaries[i].Hash)
			response.Error(c, response.ErrAgentBinariesInvalidData, err)
			return
		}
	}

	response.Success(c, http.StatusOK, resp)
}

// GetAgentBinaryFile is a function to return agent binary file
// @Summary Retrieve agent binary file by OS and arch
// @Tags Binaries
// @Produce octet-stream,json
// @Param os path string true "agent info OS" default(linux) Enums(windows, linux, darwin)
// @Param arch path string true "agent info arch" default(amd64) Enums(386, amd64)
// @Param version path string true "agent version string according semantic version format" default(latest)
// @Success 200 {file} file "agent binary as a file"
// @Failure 400 {object} utils.errorResp "invalid agent info"
// @Failure 403 {object} utils.errorResp "getting agent binary file not permitted"
// @Failure 404 {object} utils.errorResp "agent binary file not found"
// @Failure 500 {object} utils.errorResp "internal error on getting agent binary file"
// @Router /binaries/vxagent/{os}/{arch}/{version} [get]
func (s *BinariesService) GetAgentBinaryFile(c *gin.Context) {
	var (
		agentOS      = c.Param("os")
		agentArch    = c.Param("arch")
		agentVersion = c.Param("version")
		agentName    = "vxagent"
		binary       models.Binary
		chksums      models.BinaryChksum
		data         []byte
		ok           bool
		resultName   string
		s3           storage.IStorage
		validate     = validator.New()
	)
	uaf := useraction.NewFields(c, "agent", "distribution", "downloading", "", useraction.UnknownObjectDisplayName)
	defer s.userActionWriter.WriteUserAction(c, uaf)

	resultName = fmt.Sprintf("%s_%s_%s", agentName, agentOS, agentArch)
	if agentOS == "windows" {
		agentName += ".exe"
		resultName += ".exe"
	}
	uaf.ObjectDisplayName = resultName

	if err := validate.Var(agentOS, "oneof=windows linux darwin,required"); err != nil {
		response.Error(c, response.ErrAgentBinaryFileInvalidOS, err)
		return
	}
	if err := validate.Var(agentArch, "oneof=386 amd64,required"); err != nil {
		response.Error(c, response.ErrAgentBinaryFileInvalidArch, err)
		return
	}
	if err := validate.Var(agentVersion, "max=25,required"); err != nil {
		response.Error(c, response.ErrAgentBinaryFileInvalidArch, err)
		return
	}

	tid, _ := context.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("tenant_id IN (?)", []uint64{0, tid}).Where("type LIKE ?", "vxagent")
		if agentVersion == "latest" {
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC, ver_build DESC")
		} else {
			return db.Where("version LIKE ?", agentVersion)
		}
	}

	err := s.db.Scopes(scope).Model(&binary).Take(&binary).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		logger.FromContext(c).Errorf("error getting binary info by version '%s', record not found", agentVersion)
		response.Error(c, response.ErrAgentBinaryFileNotFound, err)
		return
	} else if err != nil {
		logger.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", agentVersion)
		response.Error(c, response.ErrInternal, err)
		return
	}
	uaf.ObjectID = binary.Hash

	path := filepath.Join("vxagent", binary.Version, agentOS, agentArch, agentName)
	if chksums, ok = binary.Info.Chksums[path]; !ok {
		logger.FromContext(c).Errorf("error getting agent binary file check sums: '%s' not found", path)
		response.Error(c, response.ErrAgentBinaryFileNotFound, nil)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		response.Error(c, response.ErrInternal, err)
		return
	}

	if data, err = s3.ReadFile(path); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error reading agent binary file '%s'", path)
		response.Error(c, response.ErrInternal, err)
		return
	}

	if err = validateBinaryFileByChksums(data, chksums); err != nil {
		logger.FromContext(c).WithError(err).Errorf("error validating agent binary file by check sums '%s'", path)
		response.Error(c, response.ErrAgentBinaryFileCorrupted, err)
		return
	}

	uaf.Success = true
	c.Set("uaf", []useraction.Fields{uaf})
	c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resultName))
	c.Data(http.StatusOK, "application/octet-stream", data)
}
