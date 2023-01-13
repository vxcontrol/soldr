package utils

import (
	"errors"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
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
