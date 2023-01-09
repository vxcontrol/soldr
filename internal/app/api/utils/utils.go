package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/qor/validations"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	obs "soldr/internal/observability"
	"soldr/internal/storage"
	"soldr/internal/system"
	"soldr/internal/version"

	"soldr/internal/app/api/models"
	"soldr/internal/app/api/utils/meter"
)

// Constant enum as a return code from compare two semantic version
const (
	CompareVersionError = iota - 4
	SourceVersionInvalid
	SourceVersionEmpty
	SourceVersionGreat
	VersionsEqual
	TargetVersionGreat
	TargetVersionEmpty
	TargetVersionInvalid
)

const DefaultSessionTimeout int = 3 * 3600 // 3 hours

const (
	// URL prefix for each REST API endpoint
	PrefixPathAPI = "/api/v1"
)

type UserActionFields struct {
	Domain            string
	ObjectType        string
	ObjectId          string
	ObjectDisplayName string
	ActionCode        string
	Success           bool
	FailReason        string
}

const UnknownObjectDisplayName = "Undefined object"

func GetAgentName(c *gin.Context, hash string) (string, error) {
	iDB := GetGormDB(c, "iDB")
	if iDB == nil {
		return "", errors.New("can't connect to database")
	}
	var agent models.Agent
	if err := iDB.Take(&agent, "hash = ?", hash).Error; err != nil {
		return "", err
	}
	return agent.Description, nil
}

func GetGroupName(c *gin.Context, hash string) (string, error) {
	iDB := GetGormDB(c, "iDB")
	if iDB == nil {
		return "", errors.New("can't connect to database")
	}
	var group models.Group
	if err := iDB.Take(&group, "hash = ?", hash).Error; err != nil {
		return "", err
	}
	return group.Info.Name.En, nil
}

type GroupedData struct {
	Grouped []string `json:"grouped"`
	Total   uint64   `json:"total"`
}

//lint:ignore U1000 successResp
type successResp struct {
	Status string      `json:"status" example:"success"`
	Data   interface{} `json:"data" swaggertype:"object"`
} // @name SuccessResponse

//lint:ignore U1000 errorResp
type errorResp struct {
	Status  string `json:"status" example:"error"`
	Code    string `json:"code" example:"Internal"`
	Msg     string `json:"msg,omitempty" example:"internal server error"`
	Error   string `json:"error,omitempty" example:"original server error message"`
	TraceID string `json:"trace_id,omitempty" example:"1234567890abcdef1234567890abcdef"`
} // @name ErrorResponse

// IsUseSSL is function to return true if server configured to use TLS for incoming connections
func IsUseSSL() bool {
	return os.Getenv("API_USE_SSL") == "true"
}

// FromContext is function to get logrus Entry with context
func FromContext(c *gin.Context) *logrus.Entry {
	return logrus.WithContext(c.Request.Context())
}

// UniqueUint64InSlice is function to remove duplicates in slice of uint64
func UniqueUint64InSlice(slice []uint64) []uint64 {
	keys := make(map[uint64]bool)
	list := []uint64{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// StringInSlice is function to lookup string value in slice of strings
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// StringsInSlice is function to lookup all strings in slice of strings
func StringsInSlice(a []string, list []string) bool {
	for _, b := range a {
		if !StringInSlice(b, list) {
			return false
		}
	}
	return true
}

// GetDB is function to make GORM DB connection
// TODO: return error
// Do not use anywhere except main(), pass db as a dependency for service.
// For agent server DB use mem.ServiceDBConnectionStorage.
// Deprecated
func GetDB(user, pass, host, port, name string) *gorm.DB {
	addr := fmt.Sprintf("%s:%s@%s/%s?parseTime=true",
		user, pass, fmt.Sprintf("tcp(%s:%s)", host, port), name)

	conn, err := gorm.Open("mysql", addr)
	if err != nil {
		logrus.WithField("component", "gorm_conn_getter").
			Errorf("error opening gorm connection: %v", err)
		return nil
	}
	conn.LogMode(true) // avoid annoying logs
	validations.RegisterCallbacks(conn)

	conn.DB().SetMaxIdleConns(10)
	conn.DB().SetMaxOpenConns(100)
	conn.DB().SetConnMaxLifetime(time.Hour)

	meter.ApplyGorm(conn)

	// conn.SetLogger(logrus.StandardLogger())
	return conn
}

// ApplyToChainDB is function to extend gorm method chaining by custom functions
func ApplyToChainDB(db *gorm.DB, funcs ...func(*gorm.DB) *gorm.DB) (tx *gorm.DB) {
	for _, f := range funcs {
		db = f(db)
	}
	return db
}

// EncryptPassword is function to prepare user data as a password
func EncryptPassword(password string) (hpass []byte, err error) {
	hpass, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return
}

// ActionsMapper is function to make action field mapper for modules table
func ActionsMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "`{{table}}`.info", "actions")
}

// EventsMapper is function to make events field mapper for modules table
func EventsMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "`{{table}}`.info", "events")
}

// FieldsMapper is function to make fields field mapper for modules table
func FieldsMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "`{{table}}`.info", "$.fields")
}

// TagsMapper is function to make tags field mapper for common tables
// such as agents, groups, modules, policies
func TagsMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "`{{table}}`.info", "$.tags")
}

// ModulesOSMapper is function to make os type and arch field mapper for modules query results
func ModulesOSMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	sre := `%{0,1}([a-zA-Z]+):([a-zA-Z0-9]+)%{0,1}`
	scond := "JSON_SEARCH(JSON_EXTRACT(LOWER(`modules`.info), LOWER(CONCAT('$.os.', ?))), 'all', LOWER(?), NULL, '$') IS NOT NULL"
	return JsonDictMapper(q, db, value, sre, scond)
}

// ModulesOSArchMapper is function to make os arch field mapper for modules query results
func ModulesOSArchMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "JSON_EXTRACT(`modules`.info, '$.os.*[*]')", "$")
}

// ModulesOSTypeMapper is function to make os type field mapper for modules query results
func ModulesOSTypeMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "JSON_KEYS(JSON_EXTRACT(`modules`.info, '$.os'))", "$")
}

// OptionsFieldsMapper is function to make fields field mapper for options query results
func OptionsFieldsMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "`list`.config", "$.fields")
}

// OptionsOSMapper is function to make type and arch field mapper for options query results
func OptionsOSMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	sre := `%{0,1}([a-zA-Z]+):([a-zA-Z0-9]+)%{0,1}`
	scond := "JSON_SEARCH(JSON_EXTRACT(LOWER(`list`.module_os), LOWER(CONCAT('$.', ?))), 'all', LOWER(?), NULL, '$') IS NOT NULL"
	return JsonDictMapper(q, db, value, sre, scond)
}

// OptionsOSArchMapper is function to make os arch field mapper for options query results
func OptionsOSArchMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "JSON_EXTRACT(`list`.module_os, '$.*[*]')", "$")
}

// OptionsOSTypeMapper is function to make os type field mapper for options query results
func OptionsOSTypeMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	return JsonArrayMapper(q, db, value, "JSON_KEYS(JSON_EXTRACT(`list`.module_os, '$'))", "$")
}

// JsonDictMapper is function to make json dict field mapper for common tables
func JsonDictMapper(q *TableQuery, db *gorm.DB, value interface{}, sre, scond string) *gorm.DB {
	vs := make([]interface{}, 0)
	re := regexp.MustCompile(sre)
	cond := q.DoConditionFormat(scond)
	buildArray := func() *gorm.DB {
		var conds []string
		for i := 0; i < len(vs)/2; i++ {
			conds = append(conds, cond)
		}
		return db.Where(strings.Join(conds, " OR "), vs...)
	}
	parseOSArch := func(v string) []interface{} {
		list := re.FindStringSubmatch(v)
		if len(list) == 3 {
			return []interface{}{list[1], list[2]}
		}
		return []interface{}{}
	}

	switch v := value.(type) {
	case string:
		vs = append(vs, parseOSArch(v)...)
		return buildArray()
	case []interface{}:
		for _, t := range v {
			if ts, ok := t.(string); ok {
				vs = append(vs, parseOSArch(ts)...)
			}
		}
		return buildArray()
	default:
		return db
	}
}

// JsonArrayMapper is function to make json array field mapper for common tables
func JsonArrayMapper(q *TableQuery, db *gorm.DB, value interface{}, collumn, path string) *gorm.DB {
	cond := q.DoConditionFormat("JSON_SEARCH(LOWER(" + collumn + "), 'all', LOWER(?), NULL, '" + path + "') IS NOT NULL")
	buildArray := func(vs []interface{}) *gorm.DB {
		var conds []string
		for i := 0; i < len(vs); i++ {
			conds = append(conds, cond)
		}
		return db.Where(strings.Join(conds, " OR "), vs...)
	}

	switch v := value.(type) {
	case string, float64:
		return db.Where(cond, v)
	case []interface{}:
		return buildArray(v)
	default:
		return db
	}
}

// BinaryFieldMapper is function to make common fields mapper for binaries table
func BinaryFieldMapper(cond string, db *gorm.DB, value interface{}) *gorm.DB {
	switch v := value.(type) {
	case string:
		return db.Where(cond, v)
	case []string:
		var conds []string
		var vs []interface{}
		for _, t := range v {
			vs = append(vs, t)
			conds = append(conds, cond)
		}
		return db.Where(strings.Join(conds, " OR "), vs...)
	default:
		return db
	}
}

// BinaryFilesMapper is function to make files field mapper for binaries table
func BinaryFilesMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	cond := q.DoConditionFormat("JSON_SEARCH(LOWER(`{{table}}`.files), 'all', ?, NULL, '$') IS NOT NULL")
	return BinaryFieldMapper(cond, db, value)
}

// BinaryChksumsMapper is function to make chksums field mapper for binaries table
func BinaryChksumsMapper(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB {
	cond := q.DoConditionFormat("JSON_SEARCH(LOWER(`{{table}}`.chksums), 'all', ?, NULL, '$.*') IS NOT NULL")
	return BinaryFieldMapper(cond, db, value)
}

// RandBase64String is function to generate random base64 with set length (bytes)
func RandBase64String(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// MakeMD5Hash is function to generate common hash by value
func MakeMD5Hash(value, salt string) string {
	currentTime := time.Now().Format("2006-01-02 15:04:05.000000000")
	hash := md5.Sum([]byte(currentTime + value + salt))
	return hex.EncodeToString(hash[:])
}

// MakeAgentHash is function to generate agent hash from name
func MakeAgentHash(name string) string {
	return MakeMD5Hash(name, "5d3fec568afdb9b38b089265413b8901043c5889c")
}

// MakeGroupHash is function to generate group hash from name
func MakeGroupHash(name string) string {
	return MakeMD5Hash(name, "e0f6d673e27b001cae9dec148158d03d81234e3a")
}

// MakePolicyHash is function to generate policy hash from name
func MakePolicyHash(name string) string {
	return MakeMD5Hash(name, "80f184bbe13308a907fa5a6d8965953c613c8c8e")
}

// MakeServiceHash is function to generate service hash from name
func MakeServiceHash(name string) string {
	return MakeMD5Hash(name, "788058b2208248a8bdafd29e945ba1e319e65c57")
}

// MakeTenantHash is function to generate tenant hash from description
func MakeTenantHash(desc string) string {
	return MakeMD5Hash(desc, "4e55723bbe68dcd39273355ce3f90081d3b12e85")
}

// MakeUserHash is function to generate user hash from name
func MakeUserHash(name string) string {
	return MakeMD5Hash(name, "335e5be8ff97eedb86414062a94898619599b1da")
}

// MakeTaskHash is function to generate task hash
func MakeTaskHash(name string) string {
	return MakeMD5Hash(name, "e849827ddba8916293e1e52769144f78f5545986")
}

// MakeCookieStoreKey is function to generate secure key for cookie store
func MakeCookieStoreKey() []byte {
	md5Hash := func(value, salt string) string {
		hash := md5.Sum([]byte(value + "|" + salt))
		return hex.EncodeToString(hash[:])
	}
	key := strings.Join([]string{
		md5Hash(version.GetBinaryVersion(), "972bf553c89cd103feb198f62a24e305b06a8840"),
		system.MakeAgentID(),
	}, "|")
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

// MakeResultHash is function to generate result hash
func MakeResultHash(name string) string {
	return MakeMD5Hash(name, "fd872289fff6348f00085c3ca330868014359929")
}

// MakeUuidStrFromHash is function to convert format view from hash to UUID
func MakeUuidStrFromHash(hash string) (string, error) {
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}
	userIdUuid, err := uuid.FromBytes(hashBytes)
	if err != nil {
		return "", err
	}
	return userIdUuid.String(), nil
}

// GetGormDB is function to get gorm object from request context
// Do not use, pass db as a dependency for service.
// For agent server DB use mem.ServiceDBConnectionStorage.
// Deprecated
func GetGormDB(c *gin.Context, name string) *gorm.DB {
	var db *gorm.DB

	if val, ok := c.Get(name); !ok {
		logrus.WithField("component", "gorm_conn_getter").
			Errorf("error getting '" + name + "' from context")
	} else if db = val.(*gorm.DB); db == nil {
		logrus.WithField("component", "gorm_conn_getter").
			Errorf("got nil value '" + name + "' from context")
	}

	return db
}

// GetPureSemVer is function to get the most value data from semantic version string
// only major, minor and patch numbers
func GetPureSemVer(version string) string {
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	worev := strings.Split(version, "-")[0]
	parts := strings.Split(worev, ".")
	wobnum := strings.Join(parts[:min(len(parts), 3)], ".")
	return strings.Trim(wobnum, " ")
}

// CompareVersions is function to check and compare two semantic versions
// comparing mechanism is using only major, minor and patch versions
// the function may return next values:
//
//	-4 means internal error in compare mechanism
//	-3 means that sourceVersion has invalid format
//	-2 means that sourceVersion is empty
//	-1 means that sourceVersion is greater than targetVersion
//	 0 means that two versions are equal
//	 1 means that targetVersion is greater than sourceVersion
//	 2 means that targetVersion is empty
//	 3 means that targetVersion has invalid format
func CompareVersions(sourceVersion, targetVersion string) int {
	targetPureVersion := GetPureSemVer(targetVersion)
	if targetPureVersion == "" {
		return TargetVersionEmpty
	}
	targetVersionSemver, err := semver.NewVersion(targetPureVersion)
	if err != nil {
		return TargetVersionInvalid
	}

	sourcePureVersion := GetPureSemVer(sourceVersion)
	if sourcePureVersion == "" {
		return SourceVersionEmpty
	}
	sourceVersionSemver, err := semver.NewVersion(sourcePureVersion)
	if err != nil {
		return SourceVersionInvalid
	}

	comparisonValue := targetVersionSemver.Compare(sourceVersionSemver)
	switch comparisonValue {
	case -1:
		return SourceVersionGreat
	case 0:
		return VersionsEqual
	case 1:
		return TargetVersionGreat
	default:
		return CompareVersionError
	}
}

// GetInt64 is function to get some int64 value from gin context
func GetInt64(c *gin.Context, key string) (int64, bool) {
	if iv, ok := c.Get(key); !ok {
		return 0, false
	} else if v, ok := iv.(int64); !ok {
		return 0, false
	} else {
		return v, true
	}
}

// GetUint64 is function to get some uint64 value from gin context
func GetUint64(c *gin.Context, key string) (uint64, bool) {
	if iv, ok := c.Get(key); !ok {
		return 0, false
	} else if v, ok := iv.(uint64); !ok {
		return 0, false
	} else {
		return v, true
	}
}

// GetString is function to get some string value from gin context
func GetString(c *gin.Context, key string) (string, bool) {
	if iv, ok := c.Get(key); !ok {
		return "", false
	} else if v, ok := iv.(string); !ok {
		return "", false
	} else {
		return v, true
	}
}

// GetStringArray is function to get some string array value from gin context
func GetStringArray(c *gin.Context, key string) ([]string, bool) {
	if iv, ok := c.Get(key); !ok {
		return []string{}, false
	} else if v, ok := iv.([]string); !ok {
		return []string{}, false
	} else {
		return v, true
	}
}

func GetHttpClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// HTTPSuccess is function as a main part of public REST API (success response)
func HTTPSuccess(c *gin.Context, code int, data interface{}) {
	HTTPSuccessWithUAFieldsSlice(c, code, data, nil)
}

func HTTPSuccessWithUAFields(c *gin.Context, code int, data interface{}, uaFields UserActionFields) {
	HTTPSuccessWithUAFieldsSlice(c, code, data, []UserActionFields{uaFields})
}

func HTTPSuccessWithUAFieldsSlice(c *gin.Context, code int, data interface{}, uaFields []UserActionFields) {
	if !c.IsAborted() {
		if uaFields != nil {
			for i := range uaFields {
				uaFields[i].Success = true
			}
			c.Set("uaf", uaFields)
		}
		c.JSON(code, gin.H{"status": "success", "data": data})
		c.Abort()
	}
}

type HttpError interface {
	Code() string
	HttpCode() int
	Msg() string
}

// HTTPError is function as a main part of public REST API (failed response)
func HTTPError(c *gin.Context, err HttpError, original error) {
	HTTPErrorWithUAFieldsSlice(c, err, original, nil)
}

func HTTPErrorWithUAFields(c *gin.Context, err HttpError, original error, uaFields UserActionFields) {
	HTTPErrorWithUAFieldsSlice(c, err, original, []UserActionFields{uaFields})
}

func HTTPErrorWithUAFieldsSlice(c *gin.Context, err HttpError, original error, uaFields []UserActionFields) {
	if !c.IsAborted() {
		if uaFields != nil {
			for i := range uaFields {
				uaFields[i].Success = false
				uaFields[i].FailReason = err.Msg()

			}
			c.Set("uaf", uaFields)
		}
		body := gin.H{"status": "error", "code": err.Code()}

		if version.IsDevelop == "true" {
			body["msg"] = err.Msg()
			if original != nil {
				body["error"] = original.Error()
			}
		}

		traceID := obs.Observer.SpanContextFromContext(c.Request.Context()).TraceID()
		if traceID.IsValid() {
			body["trace_id"] = traceID.String()
		}

		c.JSON(err.HttpCode(), body)
		c.Abort()
	}
}

func ValidateBinaryFileByChksums(data []byte, chksums models.BinaryChksum) error {
	md5Hash := md5.Sum(data)
	if chksums.MD5 != "" && chksums.MD5 != hex.EncodeToString(md5Hash[:]) {
		return fmt.Errorf("failed to match binary file MD5 hash sum: %s", chksums.MD5)
	}

	sha256Hash := sha256.Sum256(data)
	if chksums.SHA256 != "" && chksums.SHA256 != hex.EncodeToString(sha256Hash[:]) {
		return fmt.Errorf("failed to match binary file SHA256 hash sum: %s", chksums.SHA256)
	}

	return nil
}

// UploadAgentBinariesToInstBucket is function to check and upload agent binaries to S3 instance bucket
func UploadAgentBinariesToInstBucket(binary models.Binary, iS3 storage.IStorage) error {
	joinPath := func(args ...string) string {
		tpath := filepath.Join(args...)
		return strings.Replace(tpath, "\\", "/", -1)
	}

	gS3, err := storage.NewS3(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize global S3 driver: %w", err)
	}

	prefix := joinPath("vxagent", binary.Version)
	ifiles, err := gS3.ListDirRec(prefix)
	if err != nil {
		return fmt.Errorf("failed to read info about agent binaries files: %w", err)
	}

	for _, fpath := range binary.Info.Files {
		if _, ok := ifiles[strings.TrimPrefix(fpath, prefix)]; !ok {
			return fmt.Errorf("failed to get agent binary file from global S3 '%s'", fpath)
		}
	}

	for fpath, finfo := range ifiles {
		ifpath := joinPath(prefix, fpath)
		if finfo.IsDir() || iS3.IsExist(ifpath) {
			continue
		}

		ifdata, err := gS3.ReadFile(ifpath)
		if err != nil {
			return fmt.Errorf("failed to read agent binary file '%s': %w", ifpath, err)
		}

		chksums, ok := binary.Info.Chksums[ifpath]
		if !ok {
			return fmt.Errorf("failed to get check sums of agent binary file '%s'", ifpath)
		}
		if err := ValidateBinaryFileByChksums(ifdata, chksums); err != nil {
			return fmt.Errorf("failed to check agent binary file '%s': %w", ifpath, err)
		}
		if err := iS3.WriteFile(ifpath, ifdata); err != nil {
			return fmt.Errorf("failed to write agent binary file to S3 '%s': %w", ifpath, err)
		}

		tpdata, err := json.Marshal(chksums)
		if err != nil {
			return fmt.Errorf("failed to make agent binary thumbprint for '%s': %w", ifpath, err)
		}
		tpfpath := ifpath + ".thumbprint"
		if err := iS3.WriteFile(tpfpath, tpdata); err != nil {
			return fmt.Errorf("failed to write agent binary thumbprint to S3 '%s': %w", tpfpath, err)
		}
	}

	return nil
}
