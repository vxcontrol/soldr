package storage

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/qor/validations"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"soldr/pkg/mysql"
)

// TableFilter is auxiliary struct to contain method of filtering
type TableFilter struct {
	Value interface{} `form:"value" json:"value" binding:"required" swaggertype:"object"`
	Field string      `form:"field" json:"field" binding:"required"`
}

// TableSort is auxiliary struct to contain method of sorting
type TableSort struct {
	Prop  string `form:"prop" json:"prop" binding:"omitempty"`
	Order string `form:"order" json:"order" binding:"omitempty"`
}

// TableQuery is main struct to contain input params
type TableQuery struct {
	// Number of page (since 1)
	Page int `form:"page" json:"page" binding:"min=1,required" default:"1" minimum:"1"`
	// Amount items per page (min -1, max 100, -1 means unlimited)
	Size int `form:"pageSize" json:"pageSize" binding:"min=-1,max=100,required" default:"5" minimum:"-1" maximum:"100"`
	// Type of request
	Type string `form:"type" json:"type" binding:"oneof=sort filter init page size,required" default:"init" enums:"sort,filter,init,page,size"`
	// Language of result data
	Lang string `form:"lang" json:"lang" binding:"oneof=ru en,required" default:"en" enums:"en,ru"`
	// Sorting result on server e.g. {"prop":"...","order":"..."}
	//   field order is "ascending" or "descending" value
	Sort TableSort `form:"sort" json:"sort" binding:"required" swaggertype:"string" default:"{}"`
	// Filtering result on server e.g. {"value":[...],"field":"..."}
	//   field value should be integer or string or array type
	Filters []TableFilter `form:"filters[]" json:"filters[]" binding:"omitempty" swaggertype:"array,string"`
	// Field to group results by
	Group string `form:"group" json:"group" binding:"omitempty" swaggertype:"string"`
	// non input arguments
	table      string                                        `form:"-" json:"-"`
	groupField string                                        `form:"-" json:"-"`
	sqlMappers map[string]interface{}                        `form:"-" json:"-"`
	sqlFind    func(out interface{}) func(*gorm.DB) *gorm.DB `form:"-" json:"-"`
	sqlFilters []func(*gorm.DB) *gorm.DB                     `form:"-" json:"-"`
	sqlOrders  []func(*gorm.DB) *gorm.DB                     `form:"-" json:"-"`
}

// Init is function to set table name and sql mapping to data columns
func (q *TableQuery) Init(table string, sqlMappers map[string]interface{}) error {
	q.table = table
	q.sqlFind = func(out interface{}) func(db *gorm.DB) *gorm.DB {
		return func(db *gorm.DB) *gorm.DB {
			return db.Find(out)
		}
	}
	q.sqlMappers = make(map[string]interface{})
	q.sqlOrders = append(q.sqlOrders, func(db *gorm.DB) *gorm.DB {
		return db.Order("id DESC")
	})
	for k, v := range sqlMappers {
		switch t := v.(type) {
		case string:
			t = q.DoConditionFormat(t)
			if strings.HasSuffix(t, "id") {
				q.sqlMappers[k] = t
			} else {
				q.sqlMappers[k] = "LOWER(" + t + ")"
			}
		case func(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB:
			q.sqlMappers[k] = t
		default:
			continue
		}
	}
	if q.Group != "" {
		var ok bool
		q.groupField, ok = q.sqlMappers[q.Group].(string)
		if !ok {
			return errors.New("wrong field for grouping")
		}
	}
	return nil
}

// DoConditionFormat is auxiliary function to prepare condition to the table
func (q *TableQuery) DoConditionFormat(cond string) string {
	cond = strings.ReplaceAll(cond, "{{lang}}", q.Lang)
	cond = strings.ReplaceAll(cond, "{{type}}", q.Type)
	cond = strings.ReplaceAll(cond, "{{table}}", q.table)
	cond = strings.ReplaceAll(cond, "{{page}}", strconv.Itoa(q.Page))
	cond = strings.ReplaceAll(cond, "{{size}}", strconv.Itoa(q.Size))
	return cond
}

// SetFilters is function to set custom filters to build target SQL query
func (q *TableQuery) SetFilters(sqlFilters []func(*gorm.DB) *gorm.DB) {
	q.sqlFilters = sqlFilters
}

// SetFind is function to set custom find function to build target SQL query
func (q *TableQuery) SetFind(find func(out interface{}) func(*gorm.DB) *gorm.DB) {
	q.sqlFind = find
}

// SetOrders is function to set custom ordering to build target SQL query
func (q *TableQuery) SetOrders(sqlOrders []func(*gorm.DB) *gorm.DB) {
	q.sqlOrders = sqlOrders
}

// Mappers is getter for private field (SQL find funcction to use it in custom query)
func (q *TableQuery) Find(out interface{}) func(*gorm.DB) *gorm.DB {
	return q.sqlFind(out)
}

// Mappers is getter for private field (SQL mappers fields to table ones)
func (q *TableQuery) Mappers() map[string]interface{} {
	return q.sqlMappers
}

// Table is getter for private field (table name)
func (q *TableQuery) Table() string {
	return q.table
}

// Ordering is function to get order of data rows according with input params
func (q *TableQuery) Ordering() func(db *gorm.DB) *gorm.DB {
	field := ""
	arrow := ""
	switch q.Sort.Order {
	case "ascending":
		arrow = "ASC"
	case "descending":
		arrow = "DESC"
	}
	if v, ok := q.sqlMappers[q.Sort.Prop]; ok {
		if s, ok := v.(string); ok {
			field = s
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		if field != "" && arrow != "" {
			db = db.Order(field + " " + arrow)
		}
		for _, order := range q.sqlOrders {
			db = order(db)
		}
		return db
	}
}

// Paginate is function to navigate between pages according with input params
func (q *TableQuery) Paginate() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if q.Page <= 0 && q.Size > 0 {
			return db.Limit(q.Size)
		} else if q.Page > 0 && q.Size > 0 {
			offset := (q.Page - 1) * q.Size
			return db.Offset(offset).Limit(q.Size)
		}
		return db
	}
}

// GroupBy is function to group results by some field
func (q *TableQuery) GroupBy(total *uint64, result interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Group(q.groupField).Where(q.groupField+`<> "NULL"`).Count(total).Pluck(q.groupField, result)
	}
}

// DataFilter is function to build main data filter from filters input params
func (q *TableQuery) DataFilter() func(db *gorm.DB) *gorm.DB {
	fl := make(map[string][]interface{})
	setFilter := func(field string, value interface{}) {
		fvalue := []interface{}{}
		if fv, ok := fl[field]; ok {
			fvalue = fv
		}
		switch tvalue := value.(type) {
		case string, float64, bool:
			fl[field] = append(fvalue, tvalue)
		case []interface{}:
			fl[field] = append(fvalue, tvalue)
		}
	}

	for _, f := range q.Filters {
		if _, ok := q.sqlMappers[f.Field]; ok {
			if v, ok := f.Value.(string); ok && v != "" {
				setFilter(f.Field, "%"+strings.ToLower(v)+"%")
			}
			if v, ok := f.Value.(float64); ok {
				setFilter(f.Field, v)
			}
			if v, ok := f.Value.(bool); ok {
				setFilter(f.Field, v)
			}
			if v, ok := f.Value.([]interface{}); ok && len(v) != 0 {
				var vi []interface{}
				for _, ti := range v {
					if ts, ok := ti.(string); ok {
						vi = append(vi, strings.ToLower(ts))
					}
					if ts, ok := ti.(float64); ok {
						vi = append(vi, ts)
					}
					if ts, ok := ti.(bool); ok {
						vi = append(vi, ts)
					}
				}
				if len(vi) != 0 {
					setFilter(f.Field, vi)
				}
			}
		}
	}

	return func(db *gorm.DB) *gorm.DB {
		doFilter := func(db *gorm.DB, k, s string, v interface{}) *gorm.DB {
			switch t := q.sqlMappers[k].(type) {
			case string:
				return db.Where(t+s, v)
			case func(q *TableQuery, db *gorm.DB, value interface{}) *gorm.DB:
				return t(q, db, v)
			default:
				return db
			}
		}
		for k, f := range fl {
			for _, v := range f {
				if _, ok := v.([]interface{}); ok {
					db = doFilter(db, k, " IN (?)", v)
				} else {
					db = doFilter(db, k, " LIKE ?", v)
				}
			}
		}
		for _, filter := range q.sqlFilters {
			db = filter(db)
		}
		return db
	}
}

// Query is function to retrieve table data according with input params
func (q *TableQuery) Query(db *gorm.DB, result interface{},
	funcs ...func(*gorm.DB) *gorm.DB) (uint64, error) {
	var total uint64
	err := ApplyToChainDB(
		ApplyToChainDB(db.Table(q.Table()), funcs...).Scopes(q.DataFilter()).Count(&total),
		q.Ordering(),
		q.Paginate(),
		q.Find(result),
	).Error
	return uint64(total), err
}

// QueryGrouped is function to retrieve grouped data according with input params
func (q *TableQuery) QueryGrouped(db *gorm.DB, result interface{},
	funcs ...func(*gorm.DB) *gorm.DB) (uint64, error) {
	var total uint64
	err := ApplyToChainDB(
		ApplyToChainDB(db.Table(q.Table()), funcs...).Scopes(q.DataFilter()),
		q.GroupBy(&total, result),
	).Error
	return uint64(total), err
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

	conn.SetLogger(&mysql.GormLogger{})
	if _, exists := os.LookupEnv("DEBUG"); exists {
		conn.LogMode(true)
	}

	validations.RegisterCallbacks(conn)

	conn.DB().SetMaxIdleConns(10)
	conn.DB().SetMaxOpenConns(100)
	conn.DB().SetConnMaxLifetime(time.Hour)

	mysql.ApplyGormMetrics(conn)

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
