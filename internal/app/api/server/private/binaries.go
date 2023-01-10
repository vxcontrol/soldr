package private

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"

	"soldr/internal/storage"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/utils"
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
	"files":     utils.BinaryFilesMapper,
	"chksum":    utils.BinaryChksumsMapper,
	"data": "CONCAT(`{{table}}`.version, ' | ', " +
		"`{{table}}`.hash, ' | ', " +
		"`{{table}}`.type, ' | ', " +
		"`{{table}}`.files)",
}

type BinariesService struct {
	db *gorm.DB
}

func NewBinariesService(db *gorm.DB) *BinariesService {
	return &BinariesService{
		db: db,
	}
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
		query utils.TableQuery
		resp  binaries
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrAgentBinariesInvalidRequest, err)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")

	query.Init("binaries", binariesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("tenant_id IN (0, ?)", tid).Where("type LIKE ?", "vxagent")
		},
	})

	if resp.Total, err = query.Query(s.db, &resp.Binaries); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent binaries")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	for i := 0; i < len(resp.Binaries); i++ {
		if err = resp.Binaries[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating agent binaries data '%s'", resp.Binaries[i].Hash)
			utils.HTTPError(c, srverrors.ErrAgentBinariesInvalidData, err)
			return
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
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
		err          error
		ok           bool
		resultName   string
		s3           storage.IStorage
		validate     = validator.New()
	)
	uaf := utils.UserActionFields{
		Domain:     "agent",
		ObjectType: "distribution",
		ActionCode: "downloading",
	}

	resultName = fmt.Sprintf("%s_%s_%s", agentName, agentOS, agentArch)
	if agentOS == "windows" {
		agentName += ".exe"
		resultName += ".exe"
	}
	uaf.ObjectDisplayName = resultName

	if err := validate.Var(agentOS, "oneof=windows linux darwin,required"); err != nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentBinaryFileInvalidOS, err, uaf)
		return
	}
	if err := validate.Var(agentArch, "oneof=386 amd64,required"); err != nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentBinaryFileInvalidArch, err, uaf)
		return
	}
	if err := validate.Var(agentVersion, "max=25,required"); err != nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentBinaryFileInvalidArch, err, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("tenant_id IN (?)", []uint64{0, tid}).Where("type LIKE ?", "vxagent")
		if agentVersion == "latest" {
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC, ver_build DESC")
		} else {
			return db.Where("version LIKE ?", agentVersion)
		}
	}

	err = s.db.Scopes(scope).Model(&binary).Take(&binary).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		utils.FromContext(c).WithError(nil).Errorf("error getting binary info by version '%s', record not found", agentVersion)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentBinaryFileNotFound, err, uaf)
		return
	} else if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error getting binary info by version '%s'", agentVersion)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}
	uaf.ObjectId = binary.Hash

	path := filepath.Join("vxagent", binary.Version, agentOS, agentArch, agentName)
	if chksums, ok = binary.Info.Chksums[path]; !ok {
		utils.FromContext(c).WithError(nil).Errorf("error getting agent binary file check sums: '%s' not found", path)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentBinaryFileNotFound, nil, uaf)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	if data, err = s3.ReadFile(path); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error reading agent binary file '%s'", path)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	if err = utils.ValidateBinaryFileByChksums(data, chksums); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent binary file by check sums '%s'", path)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrAgentBinaryFileCorrupted, err, uaf)
		return
	}

	uaf.Success = true
	c.Set("uaf", []utils.UserActionFields{uaf})
	c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resultName))
	c.Data(http.StatusOK, "application/octet-stream", data)
}
