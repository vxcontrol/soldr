package worker

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/internal/app/api/modules"
	"soldr/internal/crypto"
	obs "soldr/internal/observability"
	"soldr/internal/version"

	"soldr/internal/app/api/utils/dbencryptor"

	"soldr/internal/app/api/models"
	"soldr/internal/app/api/utils"
)

const (
	syncBinariesDelay  = 3 * time.Minute
	syncModulesDelay   = 30 * time.Minute
	syncRetEventsDelay = 3 * time.Hour
)

type service struct {
	iDB *gorm.DB
	sv  *models.Service
	bns []models.Binary
	nbl int
}

// TODO: use mem.ServiceDBConnectionStorage
// Deprecated
func loadServices(gDB *gorm.DB, mSV map[uint64]*service) map[uint64]*service {
	var svs []models.Service

	if err := gDB.Find(&svs).Error; err != nil {
		return mSV
	}

	for idx := range svs {
		sv := svs[idx]
		if _, ok := mSV[sv.ID]; ok {
			continue
		}
		iDB := utils.GetDB(sv.Info.DB.User, sv.Info.DB.Pass, sv.Info.DB.Host,
			strconv.Itoa(int(sv.Info.DB.Port)), sv.Info.DB.Name)
		if iDB != nil {
			mSV[sv.ID] = &service{
				iDB: iDB,
				sv:  &sv,
			}
		}
	}

	return mSV
}

func loadModules(ctx context.Context, gDB *gorm.DB) []*models.ModuleS {
	var (
		lmods []*models.ModuleS
		rmods []*models.ModuleS
	)

	if err := gDB.Find(&lmods, "state LIKE 'release' AND service_type LIKE 'vxmonitor'").Error; err != nil {
		logrus.WithContext(ctx).WithError(err).Error("failed to load modules from global registry")
		return lmods
	}

	for _, mod := range lmods {
		if err := mod.Valid(); err != nil {
			logrus.WithContext(ctx).WithError(err).
				Errorf("failed to validate module '%s': '%s'", mod.Info.Name, mod.Info.Version.String())
			continue
		}
		rmods = append(rmods, mod)
	}

	return rmods
}

func checkIsModuleNeedUpdate(ctx context.Context, gDB *gorm.DB, moduleS *models.ModuleS, srv *service) bool {
	var (
		count int64
		mod   models.ModuleA
	)

	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ? AND version LIKE ? AND last_module_update NOT LIKE ?",
			moduleS.Info.Name, moduleS.Info.Version.String(), moduleS.LastUpdate)
	}
	if err := srv.iDB.Scopes(scope).Model(&mod).Count(&count).Error; err != nil {
		logrus.WithContext(ctx).WithError(err).
			Warnf("error counting instance modules by name and version '%s' '%s'",
				moduleS.Info.Name, moduleS.Info.Version.String())
		return false
	}

	return count != 0
}

func updateModuleInPolicies(ctx context.Context, encryptor crypto.IDBConfigEncryptor, moduleS *models.ModuleS, srv *service) {
	logrus.WithContext(ctx).Infof("want to update module '%s' '%s'",
		moduleS.Info.Name, moduleS.Info.Version.String())

	var modulesA []models.ModuleA
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ? AND version LIKE ? AND last_module_update NOT LIKE ?",
			moduleS.Info.Name, moduleS.Info.Version.String(), moduleS.LastUpdate)
	}
	if err := srv.iDB.Scopes(scope).Find(&modulesA).Error; err != nil {
		logrus.WithContext(ctx).WithError(err).
			Warnf("error finding policy modules by name and version '%s' '%s'",
				moduleS.Info.Name, moduleS.Info.Version.String())
		return
	} else if len(modulesA) == 0 {
		return
	}

	if err := modules.CopyModuleAFilesToInstanceS3(&moduleS.Info, srv.sv); err != nil {
		logrus.WithContext(ctx).WithError(err).Warnf("error copying module files to S3")
		return
	}

	excl := []string{"policy_id", "status", "join_date", "last_update"}
	var err error
	for _, moduleA := range modulesA {
		moduleA, err = modules.MergeModuleAConfigFromModuleS(&moduleA, moduleS, encryptor)
		if err != nil {
			if _, ok := err.(*crypto.ErrDecryptFailed); ok || err != nil {
				continue
			}
		}
		if err = srv.iDB.Omit(excl...).Save(&moduleA).Error; err != nil {
			logrus.WithContext(ctx).WithError(err).Warnf("error updating module: error saving module data")
		}
	}

	logrus.WithContext(ctx).Infof("update module '%s' '%s' was done",
		moduleS.Info.Name, moduleS.Info.Version.String())
}

func getRunningExecutablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate the symbolic link to get the real path: %w", err)
	}
	return path, nil
}

func getBinHashes() (string, string, error) {
	execFile, err := getRunningExecutablePath()
	if err != nil {
		return "", "", fmt.Errorf("failed to get the current executable path: %w", err)
	}
	f, err := os.Open(execFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to open the executable file %s: %w", execFile, err)
	}
	defer f.Close()

	hSHA256 := sha256.New()
	f.Seek(0, io.SeekStart)
	if _, err := io.Copy(hSHA256, f); err != nil {
		return "", "", fmt.Errorf("failed to get the sha256 hash of the executable file: %w", err)
	}
	sha256Hash := hSHA256.Sum(nil)

	hMD5 := md5.New()
	f.Seek(0, io.SeekStart)
	if _, err := io.Copy(hMD5, f); err != nil {
		return "", "", fmt.Errorf("failed to get the md5 hash of the executable file: %w", err)
	}
	md5Hash := hMD5.Sum(nil)

	return hex.EncodeToString(md5Hash[:]), hex.EncodeToString(sha256Hash[:]), nil
}

func getBinVersion() (models.BinaryVersion, string) {
	var binVersion models.BinaryVersion
	binVersionString := version.GetBinaryVersion()
	rgxSemVer := regexp.MustCompile(`^(v)?([0-9]+)\.([0-9]+)(\.([0-9]+))?(\.([0-9]+))?(-([a-zA-Z0-9]+))?$`)
	strParts := rgxSemVer.FindAllStringSubmatch(binVersionString, 3)
	if len(strParts) != 1 {
		return binVersion, binVersionString
	}
	strToUint64 := func(str string) uint64 {
		if i, err := strconv.Atoi(str); err != nil {
			return 0
		} else {
			return uint64(i)
		}
	}
	if len(strParts[0]) >= 3 {
		binVersion.Major = strToUint64(strParts[0][2])
	}
	if len(strParts[0]) >= 4 {
		binVersion.Minor = strToUint64(strParts[0][3])
	}
	if len(strParts[0]) >= 6 {
		binVersion.Patch = strToUint64(strParts[0][5])
	}
	if len(strParts[0]) >= 8 {
		binVersion.Build = strToUint64(strParts[0][7])
	}
	if len(strParts[0]) >= 10 {
		binVersion.Rev = strParts[0][9]
	}
	return binVersion, binVersionString
}

func getExtConnmodels() []models.ExtConn {
	extConnVersion, extConnVersionString := getBinVersion()
	md5Hash, sha256Hash, err := getBinHashes()
	if err != nil {
		return nil
	}
	extConnInfo := models.ExtConnInfo{
		Version: extConnVersion,
		Chksums: map[string]models.BinaryChksum{
			extConnVersionString: {
				MD5:    md5Hash,
				SHA256: sha256Hash,
			},
		},
	}
	extConns := make([]models.ExtConn, 0, 2)
	for _, ctype := range []string{"browser", "external"} {
		extConns = append(extConns, models.ExtConn{
			Hash: utils.MakeMD5Hash(extConnVersionString+ctype, "ext_conns"),
			Desc: "vxapi connection",
			Type: ctype,
			Info: extConnInfo,
		})
	}
	return extConns
}

func SyncBinariesAndExtConns(ctx context.Context, gDB *gorm.DB) {
	extConns := getExtConnmodels()
	mSV := make(map[uint64]*service)

	syncBinariesToInstanceDB := func(s *service) {
		scope := func(db *gorm.DB) *gorm.DB {
			return db.
				Where("tenant_id IN (0, ?)", s.sv.TenantID).
				Where("type LIKE ?", "vxagent").
				Where("NOT ISNULL(version)")
		}
		if err := gDB.Scopes(scope).Find(&s.bns).Error; err != nil {
			return
		}
		if len(s.bns) == s.nbl {
			return
		}
		s.nbl = 0
		var count int64
		for _, bn := range s.bns {
			if err := s.iDB.Model(&bn).Count(&count).Error; err != nil {
				continue
			} else if count > 0 {
				s.nbl++
				continue
			}
			if err := s.iDB.Select("id", "hash", "info").Create(&bn).Error; err != nil {
				continue
			}
			s.nbl++
		}
	}
	syncExtConnsToInstanceDB := func(s *service) {
		for _, extConn := range extConns {
			var count int64
			ver := extConn.Info.Version
			binVersion := fmt.Sprintf("v%d.%d.%d.%d", ver.Major, ver.Minor, ver.Patch, ver.Build)
			if ver.Rev != "" {
				binVersion = fmt.Sprintf("%s-%s", binVersion, ver.Rev)
			}
			scope := func(db *gorm.DB) *gorm.DB {
				return db.Where("version LIKE ? AND type LIKE ?", binVersion, extConn.Type)
			}
			if err := s.iDB.Scopes(scope).Model(&extConn).Count(&count).Error; err != nil || count != 0 {
				s.iDB.Scopes(scope).Model(&extConn).Update(&extConn)
			} else {
				s.iDB.Create(&extConn)
			}
		}
	}
	for {
		mSV = loadServices(gDB, mSV)
		for _, s := range mSV {
			syncBinariesToInstanceDB(s)
			syncExtConnsToInstanceDB(s)
		}
		select {
		case <-time.NewTimer(syncBinariesDelay).C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func SyncModulesToPolicies(ctx context.Context, gDB *gorm.DB) {
	mSV := make(map[uint64]*service)
	encryptor := dbencryptor.NewSecureConfigEncryptor(dbencryptor.GetKey)

	syncModulesToInstance := func(ctx context.Context, mMods []*models.ModuleS, srv *service) {
		for _, mod := range mMods {
			if !checkIsModuleNeedUpdate(ctx, gDB, mod, srv) {
				continue
			}
			updateModuleInPolicies(ctx, encryptor, mod, srv)
		}
	}

	for {
		mSV = loadServices(gDB, mSV)
		ctx, span := obs.Observer.NewSpan(ctx, obs.SpanKindClient, "modules_syncer")
		mMods := loadModules(ctx, gDB)
		for _, s := range mSV {
			syncModulesToInstance(ctx, mMods, s)
		}
		span.End()
		select {
		case <-time.NewTimer(syncModulesDelay).C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func SyncRetentionEvents(ctx context.Context, gDB *gorm.DB) {
	var (
		err            error
		keepAmountDays int
	)
	if retEvents, ok := os.LookupEnv("RETENTION_EVENTS"); !ok {
		logrus.Info("events retention policy is not set")
		return
	} else if keepAmountDays, err = strconv.Atoi(retEvents); err != nil {
		logrus.WithError(err).Error("events retention policy must contains amount days")
		return
	}

	mSV := make(map[uint64]*service)
	rotateEventsInstance := func(ctx context.Context, srv *service) {
		var event models.Event
		sqlRes := srv.iDB.Delete(&event, "`date` < NOW() - INTERVAL ? DAY", keepAmountDays)
		if err := sqlRes.Error; err != nil {
			logrus.WithContext(ctx).WithError(err).
				Errorf("failed to rotate events for service '%s'", srv.sv.Hash)
		} else if sqlRes.RowsAffected != 0 {
			logrus.WithContext(ctx).
				Infof("deleted %d events from service '%s'", sqlRes.RowsAffected, srv.sv.Hash)
		}
	}

	for {
		mSV = loadServices(gDB, mSV)
		ctx, span := obs.Observer.NewSpan(ctx, obs.SpanKindClient, "events_syncer")
		for _, s := range mSV {
			rotateEventsInstance(ctx, s)
		}
		span.End()
		select {
		case <-time.NewTimer(syncRetEventsDelay).C:
			continue
		case <-ctx.Done():
			return
		}
	}
}
