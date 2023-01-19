package private

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/api/models"
	srvcontext "soldr/pkg/app/api/server/context"
	"soldr/pkg/app/api/server/response"
	useraction "soldr/pkg/app/api/user_action"
	"soldr/pkg/app/api/utils"
)

func getModuleTemplates(zipArchive *multipart.FileHeader, moduleName, version string) ([]Template, error) {
	templates := make(map[string]Template)
	putFile := func(ver, tpath, fpath string, fdata []byte) {
		template, ok := templates[ver]
		if !ok {
			template = make(Template)
			templates[ver] = template
		}
		tcont, ok := template[tpath]
		if !ok {
			tcont = make(map[string][]byte)
			template[tpath] = tcont
		}
		tcont[fpath] = fdata
	}

	zipBuffer, err := zipArchive.Open()
	if err != nil {
		return nil, err
	}
	defer zipBuffer.Close()

	zipLength, err := zipBuffer.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	zipBuffer.Seek(0, io.SeekStart)
	zipReader, err := zip.NewReader(zipBuffer, zipLength)
	if err != nil {
		return nil, err
	}

	readZipFile := func(zf *zip.File) ([]byte, error) {
		f, err := zf.Open()
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return ioutil.ReadAll(f)
	}
	exclNames := []string{"", ".DS_Store"}
	for _, zipFile := range zipReader.File {
		ppath := strings.Split(zipFile.Name, "/")
		if len(ppath) < 4 || utils.StringInSlice(ppath[len(ppath)-1], exclNames) {
			continue
		}
		if ppath[0] != moduleName {
			continue
		}
		if ppath[1] != version && version != "all" {
			continue
		}
		fileBytes, err := readZipFile(zipFile)
		if err != nil {
			return nil, err
		}

		putFile(ppath[1], ppath[2], strings.Join(ppath[3:], "/"), fileBytes)
	}

	result := make([]Template, 0, len(templates))
	for _, template := range templates {
		result = append(result, template)
	}
	return result, nil
}

type PortingService struct {
	db               *gorm.DB
	userActionWriter useraction.Writer
}

func NewPortingService(
	db *gorm.DB,
	userActionWriter useraction.Writer,
) *PortingService {
	return &PortingService{
		db:               db,
		userActionWriter: userActionWriter,
	}
}

// ExportModule is a function to export system module as a zip archive
// @Summary Export of zip archive which contains selected system module and versions
// @Tags Modules,Export
// @Produce octet-stream,json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format or 'latest' or 'all'" default(latest)
// @Success 200 {file} file "system module archive file"
// @Failure 403 {object} utils.errorResp "exporting system module content not permitted"
// @Failure 404 {object} utils.errorResp "system module or version not found"
// @Failure 500 {object} utils.errorResp "internal error on exporting system module"
// @Router /export/modules/{module_name}/versions/{version} [post]
func (s *PortingService) ExportModule(c *gin.Context) {
	var (
		err        error
		modules    []models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
	)
	uaf := useraction.Fields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "export",
		ObjectID:          moduleName,
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
	defer s.userActionWriter.WriteUserAction(uaf)

	if sv = getService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := srvcontext.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = s.db.Scopes(FilterModulesByVersion(version), scope).Find(&modules).Error; err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error finding system module by name")
		response.Error(c, response.ErrInternal, err)
		return
	} else {
		if len(modules) == 0 {
			logrus.WithContext(c).Errorf("system module by name and version not found: %s : %s", moduleName, version)
			response.Error(c, response.ErrPortingModuleNotFound, nil)
			return
		}
		for _, module := range modules {
			if err = module.Valid(); err != nil {
				logrus.WithContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
				response.Error(c, response.ErrExportInvalidModuleData, err)
				return
			}
		}
	}
	uaf.ObjectDisplayName = modules[len(modules)-1].Locale.Module["en"].Title

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)
	defer zipWriter.Close()

	for _, module := range modules {
		prefix := moduleName + "/" + module.Info.Version.String()
		template, err := LoadModuleSFromGlobalS3(&module.Info)
		if err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error loading system module files from S3")
			response.Error(c, response.ErrExportLoadFilesFail, err)
			return
		}
		config, err := BuildModuleSConfig(&module)
		if err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error building system module config")
			response.Error(c, response.ErrExportBuildConfigFail, err)
			return
		}
		template["config"] = config

		for folderName, folderContent := range template {
			for fileName, fileContent := range folderContent {
				zipFile, err := zipWriter.Create(prefix + "/" + folderName + "/" + fileName)
				if err != nil {
					logrus.WithContext(c).WithError(err).Errorf("error adding new system module file to zip")
					response.Error(c, response.ErrExportAddFileFail, err)
					return
				}

				if _, err = zipFile.Write(fileContent); err != nil {
					logrus.WithContext(c).WithError(err).Errorf("error writing system module file to zip")
					response.Error(c, response.ErrExportWriteFileFail, err)
					return
				}
			}
		}
	}
	if err = zipWriter.Close(); err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error closing system module archive")
		response.Error(c, response.ErrExportCloseArchiveFail, err)
		return
	}

	uaf.Success = true
	c.Set("uaf", []useraction.Fields{uaf})
	date := time.Now().Format("06.01.02")
	contentDisposition := fmt.Sprintf("attachment; filename=%s.v.%s.%s.zip", moduleName, version, date)
	c.Writer.Header().Add("Content-Disposition", contentDisposition)
	c.Data(http.StatusOK, "application/octet-stream", zipBuffer.Bytes())
}

// ImportModule is a function to import system module from zip archive
// @Summary Import from zip archive which contains selected system module and versions
// @Tags Modules,Import
// @Accept multipart/form-data
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format or 'all'" default(all)
// @Param rewrite query boolean true "override system module files and records in global DB" default(true)
// @Param archive formData file true "system module archive file"
// @Success 200 {object} utils.successResp "system module archive uploaded successful"
// @Failure 400 {object} utils.errorResp "bad format input system module archive"
// @Failure 403 {object} utils.errorResp "importing system module content not permitted"
// @Failure 404 {object} utils.errorResp "system module or version in archive not found"
// @Failure 500 {object} utils.errorResp "internal error on importing system module"
// @Router /import/modules/{module_name}/versions/{version} [post]
func (s *PortingService) ImportModule(c *gin.Context) {
	var (
		err        error
		modules    []models.ModuleS
		moduleName = c.Param("module_name")
		nmodules   []*models.ModuleS
		rewrite    = c.Query("rewrite") == "true"
		sv         *models.Service
		version    = c.Param("version")
	)

	uaf := useraction.Fields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "import",
		ObjectID:          moduleName,
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
	defer s.userActionWriter.WriteUserAction(uaf)

	if sv = getService(c); sv == nil {
		response.Error(c, response.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := srvcontext.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = s.db.Scopes(FilterModulesByVersion("all"), scope).Find(&modules).Error; err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error finding system module by name")
		response.Error(c, response.ErrInternal, err)
		return
	}
	getModule := func(version models.SemVersion) *models.ModuleS {
		for _, module := range modules {
			if module.Info.Version.String() == version.String() {
				return &module
			}
		}
		return nil
	}

	zipArchive, err := c.FormFile("archive")
	if err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error reading system module zip file")
		response.Error(c, response.ErrImportReadArchiveFail, err)
		return
	}

	templates, err := getModuleTemplates(zipArchive, moduleName, version)
	if err != nil {
		logrus.WithContext(c).WithError(err).Errorf("error parsing system module zip file")
		response.Error(c, response.ErrImportParseArchiveFail, err)
		return
	}
	if len(templates) == 0 {
		logrus.WithContext(c).Errorf("system module by name and version not found: %s : %s", moduleName, version)
		response.Error(c, response.ErrPortingModuleNotFound, nil)
		return
	}
	for _, template := range templates {
		module, err := LoadModuleSConfig(template["config"])
		if err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error parsing system module config from zip file")
			response.Error(c, response.ErrImportParseConfigFail, err)
			return
		}

		module.State = "release"
		module.TenantID = tid
		module.ServiceType = sv.Type
		if err = json.Unmarshal(template["config"]["info.json"], &module.Info); err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error parsing system module file info")
			response.Error(c, response.ErrImportParseFileFail, err)
			return
		}
		if err = module.Valid(); err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error validating system module data")
			response.Error(c, response.ErrImportValidateConfigFail, err)
			return
		}

		svModule := getModule(module.Info.Version)
		if svModule != nil && !rewrite {
			logrus.WithContext(c).Errorf("error overriding system module version: %s", module.Info.Version.String())
			response.Error(c, response.ErrImportOverrideNotPermitted, err)
			return
		}
		nmodules = append(nmodules, module)
	}
	uaf.ObjectDisplayName = nmodules[len(nmodules)-1].Locale.Module["en"].Title

	for idx, template := range templates {
		module := nmodules[idx]
		svModule := getModule(module.Info.Version)
		if err = StoreCleanModuleSToGlobalS3(&module.Info, template); err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error storing system module to S3")
			response.Error(c, response.ErrImportStoreS3Fail, err)
			return
		}
		if svModule != nil {
			module.ID = svModule.ID
			err = s.db.Omit("last_update").Save(module).Error
		} else {
			err = s.db.Create(module).Error
		}
		if err != nil {
			logrus.WithContext(c).WithError(err).Errorf("error storing system module to DB")
			response.Error(c, response.ErrImportStoreDBFail, err)
			return
		}
	}

	response.Success(c, http.StatusOK, struct{}{})
}
