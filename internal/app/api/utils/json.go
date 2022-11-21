package utils

import (
	"encoding/json"
	"reflect"

	"github.com/jinzhu/copier"
	"github.com/xeipuuv/gojsonreference"

	"soldr/internal/app/api/models"
)

type jsonType int

const (
	jtUnknown jsonType = iota
	jtNull
	jtBoolean
	jtString
	jtNumber
	jtInteger
	jtArray
	jtObject
)

type jsonSchemaType int

const (
	stOneOf jsonSchemaType = iota
	stAnyOf
	stAllOf
)

func getJsonType(v reflect.Value) jsonType {
	if !v.IsValid() {
		return jtNull
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return jtNull
		}
		return getJsonType(v.Elem())
	case reflect.Bool:
		return jtBoolean
	case reflect.String:
		return jtString
	case reflect.Float32, reflect.Float64:
		return jtNumber
	case reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		return jtInteger
	case reflect.Array, reflect.Slice:
		if v.IsNil() {
			return jtNull
		}
		return jtArray
	case reflect.Map:
		if v.IsNil() {
			return jtNull
		}
		return jtObject
	default:
		return jtUnknown
	}
}

func isSimpleJsonType(v reflect.Value) bool {
	t := getJsonType(v)
	return t != jtArray && t != jtObject
}

func isSimplejtArray(v reflect.Value) bool {
	for i := 0; i < v.Len(); i++ {
		if !isSimpleJsonType(v.Index(i)) {
			return false
		}
	}
	return true
}

func isMixedJsonTypesArray(v reflect.Value) bool {
	types := make(map[jsonType]struct{})
	for i := 0; i < v.Len(); i++ {
		types[getJsonType(v.Index(i))] = struct{}{}
	}
	return len(types) > 1
}

func getOriginalValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

func getInterfaceValue(v reflect.Value) interface{} {
	var i interface{}
	if v.IsValid() && v.CanInterface() {
		i = v.Interface()
	} else {
		switch getJsonType(v) {
		case jtBoolean:
			i = v.Bool()
		case jtString:
			i = v.String()
		case jtNumber:
			i = v.Float()
		case jtInteger:
			i = v.Int()
		case jtArray, jtObject:
			// case resolves via Interface
		default:
			// jtNull, jtUnknown
		}
	}
	return i
}

func deepCompareWithMerge(ov1, ov2 reflect.Value) reflect.Value {
	v1, v2 := getOriginalValue(ov1), getOriginalValue(ov2)
	tv1, tv2 := getJsonType(v1), getJsonType(v2)

	if tv1 != tv2 {
		return ov2
	}

	switch tv2 {
	case jtBoolean, jtString, jtNumber, jtInteger:
		return ov1
	case jtArray:
		if isSimplejtArray(v1) && isSimplejtArray(v2) &&
			!isMixedJsonTypesArray(v1) && !isMixedJsonTypesArray(v2) {
			if v1.Len() > 0 && v2.Len() > 0 && getJsonType(v1.Index(0)) == getJsonType(v2.Index(0)) {
				return ov1
			} else if v2.Len() == 0 {
				return ov1
			}
			return ov2
		} else {
			av := []interface{}{}
			for i := 0; i < v2.Len(); i++ {
				if i < v1.Len() {
					if rv := deepCompareWithMerge(v1.Index(i), v2.Index(i)); rv.IsValid() {
						av = append(av, getInterfaceValue(rv))
					}
				} else if !isSimpleJsonType(v2.Index(i)) {
					av = append(av, getInterfaceValue(v2.Index(i)))
				}
			}
			return reflect.ValueOf(av)
		}
	case jtObject:
		mv := reflect.MakeMap(reflect.Indirect(v2).Type())
		for _, k := range v2.MapKeys() {
			if v1.MapIndex(k).IsValid() {
				mv.SetMapIndex(k, deepCompareWithMerge(v1.MapIndex(k), v2.MapIndex(k)))
			} else {
				mv.SetMapIndex(k, v2.MapIndex(k))
			}
		}
		return mv
	default:
		// jtNull, jtUnknown
	}

	return ov2
}

func getSchemaByPath(ref gojsonreference.JsonReference, sh models.Schema) models.Type {
	var (
		ir interface{}
		rt models.Type
		rs models.Schema
	)
	copier.Copy(&rt, &sh.Type)
	rt.Ref = ""
	if b, err := json.Marshal(sh); err != nil {
		return rt
	} else if err = json.Unmarshal(b, &ir); err != nil {
		return rt
	} else if shv, _, err := ref.GetPointer().Get(ir); err != nil {
		return rt
	} else if b, err = json.Marshal(shv); err != nil {
		return rt
	} else if err = json.Unmarshal(b, &rs); err != nil {
		return rt
	}
	return rs.Type
}

func getSchemas(sh models.Schema) ([]models.Schema, jsonSchemaType) {
	var (
		shs []models.Schema
		sht jsonSchemaType = stOneOf
	)
	buildSchemas := func(shms []*models.Type) {
		for _, tv := range shms {
			if tv == nil {
				continue
			}
			tsh := models.Schema{Type: *tv}
			tsh.Definitions = sh.Definitions
			shs = append(shs, tsh)
		}
	}

	if ref, err := gojsonreference.NewJsonReference(sh.Ref); sh.Ref != "" && err == nil {
		tsh := models.Schema{Type: getSchemaByPath(ref, sh)}
		tsh.Definitions = sh.Definitions
		return getSchemas(tsh)
	} else if len(sh.AllOf) > 0 {
		sht = stAllOf
		buildSchemas(sh.AllOf)
	} else if len(sh.AnyOf) > 0 {
		sht = stAnyOf
		buildSchemas(sh.AnyOf)
	} else if len(sh.OneOf) > 0 {
		buildSchemas(sh.OneOf)
	} else {
		shs = append(shs, sh)
	}

	return shs, sht
}

func mergeArrayItems(v1, v2 reflect.Value, sh models.Schema) []interface{} {
	// build new schema to check all current items
	ash := models.Schema{Type: *sh.Items}
	ash.Definitions = sh.Definitions
	// try to clear all current items
	av := []interface{}{}
	for i := 0; i < v1.Len() && (sh.MaxItems == 0 || len(av) < sh.MaxItems); i++ {
		iv := getInterfaceValue(v1.Index(i))
		if res, err := ash.ValidateGo(iv); err == nil && res.Valid() {
			av = append(av, iv)
		} else if i < v2.Len() {
			// try fix current value via default value if it's possible
			rv := deepCompareWithMergeBySchema(v1.Index(i), v2.Index(i), ash, false)
			iv = getInterfaceValue(rv)
			if res, err := ash.ValidateGo(iv); err == nil && res.Valid() {
				av = append(av, iv)
			}
		}
	}
	// try to add items up to MinItems value
	for i := v2.Len() - 1; i >= 0 && sh.MinItems != 0 && len(av) < sh.MinItems; i-- {
		iv := getInterfaceValue(v2.Index(i))
		if res, err := ash.ValidateGo(iv); err == nil && res.Valid() {
			av = append(av, iv)
		}
	}

	return av
}

func mergeArray(ov1, ov2 reflect.Value, sh models.Schema) reflect.Value {
	v1, v2 := getOriginalValue(ov1), getOriginalValue(ov2)

	// current value is empty and it's generating error by schema
	// this items were removed explicitly or were unset previously
	if v1.Len() == 0 {
		return ov2
	}

	rv := ov1
	shs, sht := getSchemas(sh)
	for _, shm := range shs {
		v1 = getOriginalValue(rv)
		// missing schema definitions case, try to fix via merge
		if shm.Items == nil {
			tv1 := deepCompareWithMerge(rv, ov2)
			rv = deepCompareWithMergeBySchema(tv1, ov2, shm, len(shs) == 1)
		} else {
			// using schema to rebuild array items after check ones by schema
			av := mergeArrayItems(v1, v2, shm)

			// no one item passed the validation, use default value because format was changed
			if len(av) == 0 && v2.Len() > 0 {
				rv = ov2
			} else {
				// try to validate clear items list via root schema
				rv = deepCompareWithMergeBySchema(reflect.ValueOf(av), ov2, shm, len(shs) == 1)
			}
		}
		// one merged document enough
		if sht != stAllOf {
			break
		}
	}

	return rv
}

func addObjectRequiredKeys(v1, v2, mv reflect.Value, sh models.Schema) reflect.Value {
	for _, k := range v2.MapKeys() {
		kv1, kv2 := v1.MapIndex(k), v2.MapIndex(k)
		if kv2.IsValid() && StringInSlice(k.String(), sh.Required) {
			// value not found in current
			if !kv1.IsValid() {
				mv.SetMapIndex(k, kv2)
			} else if tv, ok := sh.Properties[k.String()]; ok {
				kvsh := models.Schema{Type: *tv}
				kvsh.Definitions = sh.Definitions
				// try to check original value by schema and keep it
				if res, err := kvsh.ValidateGo(getInterfaceValue(kv1)); err == nil && res.Valid() {
					mv.SetMapIndex(k, kv1)
				} else {
					mv.SetMapIndex(k, deepCompareWithMergeBySchema(kv1, kv2, kvsh, false))
				}
			} else {
				// else keep current value in temporary map
				mv.SetMapIndex(k, kv1)
			}
		}
	}

	return mv
}

func addObjectCurrentKeys(v1, v2, mv reflect.Value, sh models.Schema) reflect.Value {
	// get additionalProperties status to check it next
	aprops := true
	json.Unmarshal([]byte(sh.AdditionalProperties), &aprops)

	for _, k := range v1.MapKeys() {
		if sh.MaxProperties != 0 && len(mv.MapKeys()) >= sh.MaxProperties {
			break
		}
		// skip already added keys
		if mv.MapIndex(k).IsValid() {
			continue
		}

		kv1, kv2 := v1.MapIndex(k), v2.MapIndex(k)
		ikv1, ikv2 := getInterfaceValue(kv1), getInterfaceValue(kv2)
		// try to get schema for this key value
		if tv, ok := sh.Properties[k.String()]; ok {
			kvsh := models.Schema{Type: *tv}
			kvsh.Definitions = sh.Definitions
			// try to check original value by schema and keep it
			if res, err := kvsh.ValidateGo(ikv1); err == nil && res.Valid() {
				mv.SetMapIndex(k, kv1)
			} else if res, err := kvsh.ValidateGo(ikv2); err == nil && res.Valid() {
				mv.SetMapIndex(k, deepCompareWithMergeBySchema(kv1, kv2, kvsh, false))
			}
			// skip invalid and non required key
		} else if aprops {
			mv.SetMapIndex(k, kv1)
		}
		// else skip this key because it was removed from schema
	}

	return mv
}

func addObjectDefaultKeys(v1, v2, mv reflect.Value, sh models.Schema) reflect.Value {
	for _, k := range v2.MapKeys() {
		if sh.MinProperties == 0 || len(mv.MapKeys()) >= sh.MinProperties {
			break
		}
		// skip already added keys
		if mv.MapIndex(k).IsValid() {
			continue
		}
		mv.SetMapIndex(k, v2.MapIndex(k))
	}

	return mv
}

func mergeObject(ov1, ov2 reflect.Value, sh models.Schema) reflect.Value {
	v1, v2 := getOriginalValue(ov1), getOriginalValue(ov2)
	_ = v1

	rv := ov1
	shs, sht := getSchemas(sh)
	for _, shm := range shs {
		// make current value for new merging round
		v1 = getOriginalValue(rv)

		mv := reflect.MakeMap(reflect.Indirect(v2).Type())
		// add all new required keys from default
		mv = addObjectRequiredKeys(v1, v2, mv, shm)
		// iterate by current object properties and clear them
		mv = addObjectCurrentKeys(v1, v2, mv, shm)
		// try to add props into the object up to MinProperties value
		mv = addObjectDefaultKeys(v1, v2, mv, shm)
		// try to check result map via new merge round
		rv = deepCompareWithMergeBySchema(mv, ov2, shm, len(shs) == 1)

		// one merged document enough
		if sht != stAllOf {
			break
		}
	}

	return rv
}

func deepCompareWithMergeBySchema(ov1, ov2 reflect.Value, sh models.Schema, f bool) reflect.Value {
	if res, err := sh.ValidateGo(getInterfaceValue(ov1)); err == nil && res.Valid() {
		return ov1
	} else {
		// fallback to default value if merging was failed (option force was set)
		if f {
			return ov2
		}

		// try to fix trouble for some keys into document
		v1, v2 := getOriginalValue(ov1), getOriginalValue(ov2)
		tv1, tv2 := getJsonType(v1), getJsonType(v2)
		if tv1 != tv2 {
			return ov2
		}

		switch tv2 {
		case jtArray:
			return mergeArray(ov1, ov2, sh)
		case jtObject:
			return mergeObject(ov1, ov2, sh)
		default:
			// JSON simple types:
			// jtBoolean, jtString,
			// jtNumber, jtInteger,
			// jtNull, jtUnknown
			rv := deepCompareWithMerge(ov1, ov2)
			return deepCompareWithMergeBySchema(rv, ov2, sh, true)
		}
	}
}

// MergeTwoInterfaces is function to get origin structure of second interface
// and build new object (interface value) from first interface data
func MergeTwoInterfaces(i1, i2 interface{}) interface{} {
	rv := deepCompareWithMerge(reflect.ValueOf(i1), reflect.ValueOf(i2))
	return getInterfaceValue(rv)
}

// MergeTwoInterfacesBySchema is function to get origin structure of second interface
// and build new object (interface value) from first interface data
// BUT each iteration in recursive function to build result data is using json schema
// ** if second interface value is invalid according by schema then it'll return anyway **
func MergeTwoInterfacesBySchema(i1, i2 interface{}, sh models.Schema) interface{} {
	rv := deepCompareWithMergeBySchema(reflect.ValueOf(i1), reflect.ValueOf(i2), sh, false)
	return getInterfaceValue(rv)
}
