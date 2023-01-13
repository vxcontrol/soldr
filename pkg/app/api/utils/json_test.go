package utils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"soldr/pkg/app/api/models"
)

type jsonTest struct {
	j1, j2, jr string
}

type jsonSchemaTest struct {
	j1, j2, jr, js string
}

func checkResult(t *testing.T, j1, j2, jr string, i1, i2, ir interface{}) {
	b1, err := json.Marshal(i1)
	assert.Nilf(t, err, "i1 object was corrupted after merge: (%v)", err)
	js1 := string(b1)
	require.JSONEqf(t, j1, js1, "j1 was changed after merge: '%s' -> '%s'", j1, js1)

	b2, err := json.Marshal(i2)
	assert.Nilf(t, err, "i2 object was corrupted after merge: (%v)", err)
	js2 := string(b2)
	require.JSONEqf(t, j2, js2, "j2 was changed after merge: '%s' -> '%s'", j2, js2)

	br, err := json.Marshal(ir)
	assert.Nilf(t, err, "result object was corrupted after merge: (%v)", err)
	jsr := string(br)
	require.JSONEqf(t, jr, jsr, "unexpected json result: '%s' <-> '%s'", jr, jsr)
}

func makeTestCompare(t *testing.T, test jsonTest) {
	var (
		err        error
		i1, i2, ir interface{}
		j1, j2, jr string = test.j1, test.j2, test.jr
	)
	err = json.Unmarshal([]byte(j1), &i1)
	assert.Nilf(t, err, "bad j1 format: (%v) '%s'", err, j1)
	err = json.Unmarshal([]byte(j2), &i2)
	assert.Nilf(t, err, "bad j2 format: (%v) '%s'", err, j1)

	ir = MergeTwoInterfaces(i1, i2)
	checkResult(t, j1, j2, jr, i1, i2, ir)
}

func makeTestCompareBySchema(t *testing.T, test jsonSchemaTest) {
	var (
		err            error
		sh             models.Schema
		i1, i2, ir     interface{}
		j1, j2, jr, js string = test.j1, test.j2, test.jr, test.js
	)
	err = json.Unmarshal([]byte(j1), &i1)
	assert.Nilf(t, err, "bad j1 format: (%v) '%s'", err, j1)
	err = json.Unmarshal([]byte(j2), &i2)
	assert.Nilf(t, err, "bad j2 format: (%v) '%s'", err, j1)
	err = json.Unmarshal([]byte(js), &sh)
	assert.Nilf(t, err, "bad js format: (%v) '%s'", err, js)

	ir = MergeTwoInterfacesBySchema(i1, i2, sh)
	checkResult(t, j1, j2, jr, i1, i2, ir)
}

func TestMergeTwoJSONsSimpleTypes(t *testing.T) {
	tests := []jsonTest{
		{`true`, `false`, `true`},
		{`"new"`, `"old"`, `"new"`},
		{`123.6`, `234.3`, `123.6`},
		{`123`, `234`, `123`},
		{`null`, `null`, `null`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsNotMatchSimpleTypes(t *testing.T) {
	tests := []jsonTest{
		{`null`, `true`, `true`},
		{`null`, `"val"`, `"val"`},
		{`null`, `123.6`, `123.6`},
		{`null`, `123`, `123`},
		{`false`, `"val"`, `"val"`},
		{`false`, `123.6`, `123.6`},
		{`false`, `123`, `123`},
		{`false`, `null`, `null`},
		{`"val"`, `true`, `true`},
		{`"val"`, `123.6`, `123.6`},
		{`"val"`, `123`, `123`},
		{`"val"`, `null`, `null`},
		{`123.6`, `true`, `true`},
		{`123.6`, `"val"`, `"val"`},
		{`123.6`, `null`, `null`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsArraySimpleTypes(t *testing.T) {
	tests := []jsonTest{
		{`[true]`, `[]`, `[true]`},
		{`["new"]`, `[]`, `["new"]`},
		{`[123.6]`, `[]`, `[123.6]`},
		{`[123]`, `[]`, `[123]`},
		{`[null]`, `[]`, `[null]`},
		{`[true]`, `[false]`, `[true]`},
		{`["new"]`, `["old"]`, `["new"]`},
		{`[123.6]`, `[234.3]`, `[123.6]`},
		{`[123]`, `[234]`, `[123]`},
		{`[null]`, `[null]`, `[null]`},
		{`[true, true]`, `[false]`, `[true, true]`},
		{`["new1", "new2"]`, `["old"]`, `["new1", "new2"]`},
		{`[123.6, 123.9]`, `[234.3]`, `[123.6, 123.9]`},
		{`[123, 234]`, `[234]`, `[123, 234]`},
		{`[null, null]`, `[null]`, `[null, null]`},
		{`[true, true]`, `[false, false]`, `[true, true]`},
		{`["new1", "new2"]`, `["old1", "old2"]`, `["new1", "new2"]`},
		{`[123.6, 123.9]`, `[234.3, 234.6]`, `[123.6, 123.9]`},
		{`[123, 234]`, `[234, 345]`, `[123, 234]`},
		{`[null, null]`, `[null, null]`, `[null, null]`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsArrayMixedSimpleTypes(t *testing.T) {
	tests := []jsonTest{
		{`[true]`, `[false, null]`, `[true]`},
		{`["new"]`, `["old", null]`, `["new"]`},
		{`[123.6]`, `[234.3, null]`, `[123.6]`},
		{`[123]`, `[234, null]`, `[123]`},
		{`[null]`, `[null, false]`, `[null]`},
		{`[true, null]`, `[false]`, `[true]`},
		{`["new", null]`, `["old"]`, `["new"]`},
		{`[123.6, null]`, `[234.3]`, `[123.6]`},
		{`[123, null]`, `[234]`, `[123]`},
		{`[null, false]`, `[null]`, `[null]`},
		{`[true, "fake"]`, `[false, null]`, `[true, null]`},
		{`["new", "fake"]`, `["old", null]`, `["new", null]`},
		{`[123.6, "fake"]`, `[234.3, null]`, `[123.6, null]`},
		{`[123, "fake"]`, `[234, null]`, `[123, null]`},
		{`[null, "fake"]`, `[null, false]`, `[null, false]`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsArrayComplexTypes(t *testing.T) {
	tests := []jsonTest{
		{`[[true]]`, `[[false]]`, `[[true]]`},
		{`[["new"]]`, `[["old"]]`, `[["new"]]`},
		{`[[123.6]]`, `[[234.3]]`, `[[123.6]]`},
		{`[[123]]`, `[[234]]`, `[[123]]`},
		{`[[null]]`, `[[null]]`, `[[null]]`},
		{`[[true, true]]`, `[[false]]`, `[[true, true]]`},
		{`[["new1", "new2"]]`, `[["old"]]`, `[["new1", "new2"]]`},
		{`[[123.6, 123.9]]`, `[[234.3]]`, `[[123.6, 123.9]]`},
		{`[[123, 234]]`, `[[234]]`, `[[123, 234]]`},
		{`[[true, true]]`, `[[false, false]]`, `[[true, true]]`},
		{`[["new1", "new2"]]`, `[["old1", "old2"]]`, `[["new1", "new2"]]`},
		{`[[123.6, 123.9]]`, `[[234.3, 234.6]]`, `[[123.6, 123.9]]`},
		{`[[123, 234]]`, `[[234, 345]]`, `[[123, 234]]`},
		{`[[null, null]]`, `[[null, null]]`, `[[null, null]]`},
		{`[[true], [true]]`, `[[false]]`, `[[true]]`},
		{`[["new1"], ["new2"]]`, `[["old"]]`, `[["new1"]]`},
		{`[[123.6], [123.9]]`, `[[234.3]]`, `[[123.6]]`},
		{`[[123], [234]]`, `[[234]]`, `[[123]]`},
		{`[[null], [null]]`, `[[null]]`, `[[null]]`},
		{`[{"k":{"k":true}}]`, `[{"k":{"k":false}}]`, `[{"k":{"k":true}}]`},
		{`[{"k":{"k":"v2"}}]`, `[{"k":{"k":"v1"}}]`, `[{"k":{"k":"v2"}}]`},
		{`[{"k":{"k":123.6}}]`, `[{"k":{"k":234.3}}]`, `[{"k":{"k":123.6}}]`},
		{`[{"k":{"k":123}}]`, `[{"k":{"k":234}}]`, `[{"k":{"k":123}}]`},
		{`[{"k":{"k":null}}]`, `[{"k":{"k":null}}]`, `[{"k":{"k":null}}]`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsNotMatchComplexTypes(t *testing.T) {
	tests := []jsonTest{
		{`[[null]]`, `[[false], [false]]`, `[[false], [false]]`},
		{`[[null]]`, `[["old"], ["old"]]`, `[["old"], ["old"]]`},
		{`[[null]]`, `[[234.3], [234.6]]`, `[[234.3], [234.6]]`},
		{`[[null]]`, `[[234], [345]]`, `[[234], [345]]`},
		{`[[true]]`, `[[null], [null]]`, `[[null], [null]]`},
		{`[[null]]`, `[[false], {"k":"v"}]`, `[[false], {"k":"v"}]`},
		{`[[null]]`, `[["old"], {"k":"v"}]`, `[["old"], {"k":"v"}]`},
		{`[[null]]`, `[[234.3], {"k":"v"}]`, `[[234.3], {"k":"v"}]`},
		{`[[null]]`, `[[234], {"k":"v"}]`, `[[234], {"k":"v"}]`},
		{`[[true]]`, `[[null], {"k":"v"}]`, `[[null], {"k":"v"}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":false}]`, `[{"k":false}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":"v"}]`, `[{"k":"v"}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":234.6}]`, `[{"k":234.6}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":234}]`, `[{"k":234}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":null}]`, `[{"k":null}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":{"k":false}}]`, `[{"k":{"k":false}}]`},
		{`[{"k":{"k":123}}]`, `[{"k":{"k":"v"}}]`, `[{"k":{"k":"v"}}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":{"k":234.6}}]`, `[{"k":{"k":234.6}}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":{"k":234}}]`, `[{"k":{"k":234}}]`},
		{`[{"k":{"k":"v"}}]`, `[{"k":{"k":null}}]`, `[{"k":{"k":null}}]`},
		{`{"k":{"k":false}}`, `{"k":[true]}`, `{"k":[true]}`},
		{`{"k":{"k":"v"}}`, `{"k":["v"]}`, `{"k":["v"]}`},
		{`{"k":{"k":123.3}}`, `{"k":[234.3]}`, `{"k":[234.3]}`},
		{`{"k":{"k":123}}`, `{"k":[234]}`, `{"k":[234]}`},
		{`{"k":{"k":null}}`, `{"k":[null]}`, `{"k":[null]}`},
		{`{"k":[true]}`, `{"k":{"k":false}}`, `{"k":{"k":false}}`},
		{`{"k":["v"]}`, `{"k":{"k":"v"}}`, `{"k":{"k":"v"}}`},
		{`{"k":[234.3]}`, `{"k":{"k":123.3}}`, `{"k":{"k":123.3}}`},
		{`{"k":[234]}`, `{"k":{"k":123}}`, `{"k":{"k":123}}`},
		{`{"k":[null]}`, `{"k":{"k":null}}`, `{"k":{"k":null}}`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsObjectSimpleTypes(t *testing.T) {
	tests := []jsonTest{
		{`{"k":false}`, `{"k":true}`, `{"k":false}`},
		{`{"k":"v2"}`, `{"k":"v1"}`, `{"k":"v2"}`},
		{`{"k":123.3}`, `{"k":234.3}`, `{"k":123.3}`},
		{`{"k":123}`, `{"k":234}`, `{"k":123}`},
		{`{"k":null}`, `{"k":null}`, `{"k":null}`},
		{`{"k1":false}`, `{"k1":true,"k2":"v2"}`, `{"k1":false,"k2":"v2"}`},
		{`{"k1":"v2"}`, `{"k1":"v1","k2":"v2"}`, `{"k1":"v2","k2":"v2"}`},
		{`{"k1":123.3}`, `{"k1":234.3,"k2":"v2"}`, `{"k1":123.3,"k2":"v2"}`},
		{`{"k1":123}`, `{"k1":234,"k2":"v2"}`, `{"k1":123,"k2":"v2"}`},
		{`{"k1":null}`, `{"k1":null,"k2":"v2"}`, `{"k1":null,"k2":"v2"}`},
		{`{"k1":false,"k2":"v2"}`, `{"k1":true}`, `{"k1":false}`},
		{`{"k1":"v2","k2":"v2"}`, `{"k1":"v1"}`, `{"k1":"v2"}`},
		{`{"k1":123.3,"k2":"v2"}`, `{"k1":234.3}`, `{"k1":123.3}`},
		{`{"k1":123,"k2":"v2"}`, `{"k1":234}`, `{"k1":123}`},
		{`{"k1":null,"k2":"v2"}`, `{"k1":null}`, `{"k1":null}`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}

func TestMergeTwoJSONsObjectComplexTypes(t *testing.T) {
	tests := []jsonTest{
		{`{"k":[false]}`, `{"k":[true]}`, `{"k":[false]}`},
		{`{"k":["v2"]}`, `{"k":["v1"]}`, `{"k":["v2"]}`},
		{`{"k":[123.3]}`, `{"k":[234.3]}`, `{"k":[123.3]}`},
		{`{"k":[123]}`, `{"k":[234]}`, `{"k":[123]}`},
		{`{"k":[null]}`, `{"k":[null]}`, `{"k":[null]}`},
		{`{"k":[{"k":false}]}`, `{"k":[{"k":true}]}`, `{"k":[{"k":false}]}`},
		{`{"k":[{"k":"v2"}]}`, `{"k":[{"k":"v1"}]}`, `{"k":[{"k":"v2"}]}`},
		{`{"k":[{"k":123.3}]}`, `{"k":[{"k":234.3}]}`, `{"k":[{"k":123.3}]}`},
		{`{"k":[{"k":123}]}`, `{"k":[{"k":234}]}`, `{"k":[{"k":123}]}`},
		{`{"k":[{"k":null}]}`, `{"k":[{"k":null}]}`, `{"k":[{"k":null}]}`},
		{`{"k":{"k":false}}`, `{"k":{"k":true}}`, `{"k":{"k":false}}`},
		{`{"k":{"k":"v2"}}`, `{"k":{"k":"v1"}}`, `{"k":{"k":"v2"}}`},
		{`{"k":{"k":123.3}}`, `{"k":{"k":234.3}}`, `{"k":{"k":123.3}}`},
		{`{"k":{"k":123}}`, `{"k":{"k":234}}`, `{"k":{"k":123}}`},
		{`{"k":{"k":null}}`, `{"k":{"k":null}}`, `{"k":{"k":null}}`},
		{`{"k1":[false]}`, `{"k1":[true],"k2":["v2"]}`, `{"k1":[false],"k2":["v2"]}`},
		{`{"k1":["v2"]}`, `{"k1":["v1"],"k2":["v2"]}`, `{"k1":["v2"],"k2":["v2"]}`},
		{`{"k1":[123.3]}`, `{"k1":[234.3],"k2":["v2"]}`, `{"k1":[123.3],"k2":["v2"]}`},
		{`{"k1":[123]}`, `{"k1":[234],"k2":["v2"]}`, `{"k1":[123],"k2":["v2"]}`},
		{`{"k1":[null]}`, `{"k1":[null],"k2":["v2"]}`, `{"k1":[null],"k2":["v2"]}`},
		{`{"k1":[false],"k2":["v2"]}`, `{"k1":[true]}`, `{"k1":[false]}`},
		{`{"k1":["v2"],"k2":["v2"]}`, `{"k1":["v1"]}`, `{"k1":["v2"]}`},
		{`{"k1":[123.3],"k2":["v2"]}`, `{"k1":[234.3]}`, `{"k1":[123.3]}`},
		{`{"k1":[123],"k2":["v2"]}`, `{"k1":[234]}`, `{"k1":[123]}`},
		{`{"k1":[null],"k2":["v2"]}`, `{"k1":[null]}`, `{"k1":[null]}`},
	}
	for _, test := range tests {
		makeTestCompare(t, test)
	}
}
func TestMergeTwoNativeGolangTypes(t *testing.T) {
	require.Equal(t, false, MergeTwoInterfaces(false, true), "boolean")
	require.Equal(t, "v2", MergeTwoInterfaces("v2", "v1"), "string")
	require.Equal(t, 123.3, MergeTwoInterfaces(123.3, 234.3), "number")
	require.Equal(t, 123, MergeTwoInterfaces(123, 234), "integer")
	require.Equal(t, nil, MergeTwoInterfaces(nil, nil), "null")

	require.Equal(t, []bool{false},
		MergeTwoInterfaces([]bool{false}, []bool{true}), "array of boolean")
	require.Equal(t, []string{"v2"},
		MergeTwoInterfaces([]string{"v2"}, []string{"v1"}), "array of string")
	require.Equal(t, []float64{123.3},
		MergeTwoInterfaces([]float64{123.3}, []float64{234.3}), "array of number")
	require.Equal(t, []int64{123},
		MergeTwoInterfaces([]int64{123}, []int64{234}), "array of integer")
	require.Equal(t, []interface{}{nil},
		MergeTwoInterfaces([]interface{}{nil}, []interface{}{nil}), "array of null")

	require.Equal(t, map[string]bool{"k": false},
		MergeTwoInterfaces(map[string]bool{"k": false}, map[string]bool{"k": true}),
		"map of boolean value")
	require.Equal(t, map[string]string{"k": "v2"},
		MergeTwoInterfaces(map[string]string{"k": "v2"}, map[string]string{"k": "v1"}),
		"map of string value")
	require.Equal(t, map[string]float64{"k": 123.3},
		MergeTwoInterfaces(map[string]float64{"k": 123.3}, map[string]float64{"k": 234.3}),
		"map of number value")
	require.Equal(t, map[string]int64{"k": 123},
		MergeTwoInterfaces(map[string]int64{"k": 123}, map[string]int64{"k": 234}),
		"map of integer value")
	require.Equal(t, map[string]interface{}{"k": nil},
		MergeTwoInterfaces(map[string]interface{}{"k": nil}, map[string]interface{}{"k": nil}),
		"map of null value")
}

func TestMergeTwoJSONsBySchema(t *testing.T) {
	tests := []jsonSchemaTest{
		{`true`, `false`, `true`, `{"type": "boolean"}`},
		{`"new"`, `"old"`, `"new"`, `{"type": "string"}`},
		{`123.6`, `234.3`, `123.6`, `{"type": "number"}`},
		{`123`, `234`, `123`, `{"type": "integer"}`},
		{`null`, `null`, `null`, `{"type": "null"}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaNotMatchSimpleTypes(t *testing.T) {
	tests := []jsonSchemaTest{
		{`null`, `true`, `true`, `{"type": "boolean"}`},
		{`null`, `"val"`, `"val"`, `{"type": "string"}`},
		{`null`, `123.6`, `123.6`, `{"type": "number"}`},
		{`null`, `123`, `123`, `{"type": "integer"}`},
		{`false`, `"val"`, `"val"`, `{"type": "string"}`},
		{`false`, `123.6`, `123.6`, `{"type": "number"}`},
		{`false`, `123`, `123`, `{"type": "integer"}`},
		{`false`, `null`, `null`, `{"type": "null"}`},
		{`"val"`, `true`, `true`, `{"type": "boolean"}`},
		{`"val"`, `123.6`, `123.6`, `{"type": "number"}`},
		{`"val"`, `123`, `123`, `{"type": "integer"}`},
		{`"val"`, `null`, `null`, `{"type": "null"}`},
		{`123.6`, `true`, `true`, `{"type": "boolean"}`},
		{`123.6`, `"val"`, `"val"`, `{"type": "string"}`},
		{`123.6`, `null`, `null`, `{"type": "null"}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaArraySimpleTypes(t *testing.T) {
	tests := []jsonSchemaTest{
		{`[true]`, `[]`, `[true]`,
			`{"type": "array", "items": {"type": "boolean"}}`},
		{`["new"]`, `[]`, `["new"]`,
			`{"type": "array", "items": {"type": "string"}}`},
		{`[123.6]`, `[]`, `[123.6]`,
			`{"type": "array", "items": {"type": "number"}}`},
		{`[123]`, `[]`, `[123]`,
			`{"type": "array", "items": {"type": "integer"}}`},
		{`[null]`, `[]`, `[null]`,
			`{"type": "array", "items": {"type": "null"}}`},
		{`[true]`, `[false]`, `[true]`,
			`{"type": "array", "items": {"type": "boolean"}}`},
		{`["new"]`, `["old"]`, `["new"]`,
			`{"type": "array", "items": {"type": "string"}}`},
		{`[123.6]`, `[234.3]`, `[123.6]`,
			`{"type": "array", "items": {"type": "number"}}`},
		{`[123]`, `[234]`, `[123]`,
			`{"type": "array", "items": {"type": "integer"}}`},
		{`[null]`, `[null]`, `[null]`,
			`{"type": "array", "items": {"type": "null"}}`},
		{`[true, true]`, `[false]`, `[true, true]`,
			`{"type": "array", "items": {"type": "boolean"}}`},
		{`["new1", "new2"]`, `["old"]`, `["new1", "new2"]`,
			`{"type": "array", "items": {"type": "string"}}`},
		{`[123.6, 123.9]`, `[234.3]`, `[123.6, 123.9]`,
			`{"type": "array", "items": {"type": "number"}}`},
		{`[123, 234]`, `[234]`, `[123, 234]`,
			`{"type": "array", "items": {"type": "integer"}}`},
		{`[null, null]`, `[null]`, `[null, null]`,
			`{"type": "array", "items": {"type": "null"}}`},
		{`[true, true]`, `[false, false]`, `[true, true]`,
			`{"type": "array", "items": {"type": "boolean"}}`},
		{`["new1", "new2"]`, `["old1", "old2"]`, `["new1", "new2"]`,
			`{"type": "array", "items": {"type": "string"}}`},
		{`[123.6, 123.9]`, `[234.3, 234.6]`, `[123.6, 123.9]`,
			`{"type": "array", "items": {"type": "number"}}`},
		{`[123, 234]`, `[234, 345]`, `[123, 234]`,
			`{"type": "array", "items": {"type": "integer"}}`},
		{`[null, null]`, `[null, null]`, `[null, null]`,
			`{"type": "array", "items": {"type": "null"}}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaNotMatchArraySimpleTypes(t *testing.T) {
	tests := []jsonSchemaTest{
		{`[null]`, `[false]`, `[false]`,
			`{"type": "array", "items": {"type": "boolean"}}`},
		{`[null]`, `["old"]`, `["old"]`,
			`{"type": "array", "items": {"type": "string"}}`},
		{`[null]`, `[234.3]`, `[234.3]`,
			`{"type": "array", "items": {"type": "number"}}`},
		{`[null]`, `[234]`, `[234]`,
			`{"type": "array", "items": {"type": "integer"}}`},
		{`[false]`, `[null]`, `[null]`,
			`{"type": "array", "items": {"type": "null"}}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaArrayComplexTypes(t *testing.T) {
	tests := []jsonSchemaTest{
		{`[[true]]`, `[[false]]`, `[[true]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "boolean"}}}`},
		{`[["new"]]`, `[["old"]]`, `[["new"]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "string"}}}`},
		{`[[123.6]]`, `[[234.3]]`, `[[123.6]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "number"}}}`},
		{`[[123]]`, `[[234]]`, `[[123]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "integer"}}}`},
		{`[[null]]`, `[[null]]`, `[[null]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "null"}}}`},
		{`[[true, true]]`, `[[false]]`, `[[true, true]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "boolean"}}}`},
		{`[["new1", "new2"]]`, `[["old"]]`, `[["new1", "new2"]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "string"}}}`},
		{`[[123.6, 123.9]]`, `[[234.3]]`, `[[123.6, 123.9]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "number"}}}`},
		{`[[123, 234]]`, `[[234]]`, `[[123, 234]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "integer"}}}`},
		{`[[true, true]]`, `[[false, false]]`, `[[true, true]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "boolean"}}}`},
		{`[["new1", "new2"]]`, `[["old1", "old2"]]`, `[["new1", "new2"]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "string"}}}`},
		{`[[123.6, 123.9]]`, `[[234.3, 234.6]]`, `[[123.6, 123.9]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "number"}}}`},
		{`[[123, 234]]`, `[[234, 345]]`, `[[123, 234]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "integer"}}}`},
		{`[[null, null]]`, `[[null, null]]`, `[[null, null]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "null"}}}`},
		{`[[true], [true]]`, `[[false]]`, `[[true], [true]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "boolean"}}}`},
		{`[["new1"], ["new2"]]`, `[["old"]]`, `[["new1"], ["new2"]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "string"}}}`},
		{`[[123.6], [123.9]]`, `[[234.3]]`, `[[123.6], [123.9]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "number"}}}`},
		{`[[123], [234]]`, `[[234]]`, `[[123], [234]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "integer"}}}`},
		{`[[null], [null]]`, `[[null]]`, `[[null], [null]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "null"}}}`},
		{`[{"k":{"k":true}}]`, `[{"k":{"k":false}}]`, `[{"k":{"k":true}}]`,
			`{"type": "array", "items": {"type": "object", "properties": {"k": {"type": "object"}}}}`},
		{`[{"k":{"k":"v2"}}]`, `[{"k":{"k":"v1"}}]`, `[{"k":{"k":"v2"}}]`,
			`{"type": "array", "items": {"type": "object", "properties": {"k": {"type": "object"}}}}`},
		{`[{"k":{"k":123.6}}]`, `[{"k":{"k":234.3}}]`, `[{"k":{"k":123.6}}]`,
			`{"type": "array", "items": {"type": "object", "properties": {"k": {"type": "object"}}}}`},
		{`[{"k":{"k":123}}]`, `[{"k":{"k":234}}]`, `[{"k":{"k":123}}]`,
			`{"type": "array", "items": {"type": "object", "properties": {"k": {"type": "object"}}}}`},
		{`[{"k":{"k":null}}]`, `[{"k":{"k":null}}]`, `[{"k":{"k":null}}]`,
			`{"type": "array", "items": {"type": "object", "properties": {"k": {"type": "object"}}}}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaNotMatchComplexTypes(t *testing.T) {
	tests := []jsonSchemaTest{
		{`[[true], [true]]`, `[[false]]`, `[[true]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "boolean"}}, "maxItems": 1}`},
		{`[["new1"], ["new2"]]`, `[["old"]]`, `[["new1"]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "string"}}, "maxItems": 1}`},
		{`[[123.6], [123.9]]`, `[[234.3]]`, `[[123.6]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "number"}}, "maxItems": 1}`},
		{`[[123], [234]]`, `[[234]]`, `[[123]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "integer"}}, "maxItems": 1}`},
		{`[[null], [null]]`, `[[null]]`, `[[null]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "null"}}, "maxItems": 1}`},
		{`[[true, true]]`, `[[false]]`, `[[true]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "boolean"}, "maxItems": 1}}`},
		{`[["new1", "new2"]]`, `[["old"]]`, `[["new1"]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "string"}, "maxItems": 1}}`},
		{`[[123.6, 123.9]]`, `[[234.3]]`, `[[123.6]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "number"}, "maxItems": 1}}`},
		{`[[123, 234]]`, `[[234]]`, `[[123]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "integer"}, "maxItems": 1}}`},
		{`[[null, null]]`, `[[null]]`, `[[null]]`,
			`{"type": "array", "items": {"type": "array", "items": {"type": "null"}, "maxItems": 1}}`},
		{`{"k":[true,false]}`, `{"k":[true,true]}`, `{"k":[true,true]}`,
			`{"type": "object", "properties": {"k": {"type": "array", "items": {"type": "boolean", "enum": [true]}, "minItems": 2}}}`},
		{`{"k":["new1","new2"]}`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`,
			`{"type": "object", "properties": {"k": {"type": "array", "items": {"type": "string", "enum": ["old1", "old2"]}}}}`},
		{`{"k":[123.6,123.9]}`, `{"k":[234.3,234.6]}`, `{"k":[234.3,234.6]}`,
			`{"type": "object", "properties": {"k": {"type": "array", "items": {"type": "number", "enum": [234.3, 234.6]}}}}`},
		{`{"k":[123,234]}`, `{"k":[234,345]}`, `{"k":[234,234]}`,
			`{"type": "object", "properties": {"k": {"type": "array", "items": {"type": "number", "enum": [234, 345]}, "minItems": 2}}}`},
		{`{"k1":true}`, `{"k1":false,"k2":false}`, `{"k1":true}`,
			`{"type": "object", "properties": {"k1": {"type": "boolean"}, "k2": {"type": "boolean"}}}`},
		{`{"k1":"new1"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1"}`,
			`{"type": "object", "properties": {"k1": {"type": "string"}, "k2": {"type": "string"}}}`},
		{`{"k1":123.6}`, `{"k1":234.3,"k2":234.6}`, `{"k1":123.6}`,
			`{"type": "object", "properties": {"k1": {"type": "number"}, "k2": {"type": "number"}}}`},
		{`{"k1":123}`, `{"k1":234,"k2":345}`, `{"k1":123}`,
			`{"type": "object", "properties": {"k1": {"type": "integer"}, "k2": {"type": "integer"}}}`},
		{`{"k1":null}`, `{"k1":null,"k2":null}`, `{"k1":null}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}, "k2": {"type": "null"}}}`},
		{`{"k1":true}`, `{"k1":false,"k2":false}`, `{"k1":true,"k2":false}`,
			`{"type": "object", "properties": {"k1": {"type": "boolean"}, "k2": {"type": "boolean"}}, "required": ["k1", "k2"]}`},
		{`{"k1":"new1"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1","k2":"old2"}`,
			`{"type": "object", "properties": {"k1": {"type": "string"}, "k2": {"type": "string"}}, "required": ["k1", "k2"]}`},
		{`{"k1":123.6}`, `{"k1":234.3,"k2":234.6}`, `{"k1":123.6,"k2":234.6}`,
			`{"type": "object", "properties": {"k1": {"type": "number"}, "k2": {"type": "number"}}, "required": ["k1", "k2"]}`},
		{`{"k1":123}`, `{"k1":234,"k2":345}`, `{"k1":123,"k2":345}`,
			`{"type": "object", "properties": {"k1": {"type": "integer"}, "k2": {"type": "integer"}}, "required": ["k1", "k2"]}`},
		{`{"k1":null}`, `{"k1":null,"k2":null}`, `{"k1":null,"k2":null}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}, "k2": {"type": "null"}}, "required": ["k1", "k2"]}`},
		{`{"k1":true,"k2":true}`, `{"k1":false,"k3":false}`, `{"k1":true,"k2":true,"k3":false}`,
			`{"type": "object", "properties": {"k1": {"type": "boolean"}}, "required": ["k1", "k3"]}`},
		{`{"k1":"new1","k2":"new2"}`, `{"k1":"old1","k3":"old3"}`, `{"k1":"new1","k2":"new2","k3":"old3"}`,
			`{"type": "object", "properties": {"k1": {"type": "string"}}, "required": ["k1", "k3"]}`},
		{`{"k1":123.6,"k2":123.9}`, `{"k1":234.3,"k3":234.6}`, `{"k1":123.6,"k2":123.9,"k3":234.6}`,
			`{"type": "object", "properties": {"k1": {"type": "number"}}, "required": ["k1", "k3"]}`},
		{`{"k1":123,"k2":345}`, `{"k1":234,"k3":456}`, `{"k1":123,"k2":345,"k3":456}`,
			`{"type": "object", "properties": {"k1": {"type": "integer"}}, "required": ["k1", "k3"]}`},
		{`{"k1":null,"k2":null}`, `{"k1":null,"k3":null}`, `{"k1":null,"k2":null,"k3":null}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}, "required": ["k1", "k3"]}`},
		{`{"k1":true,"k2":true}`, `{"k1":false}`, `{"k1":true,"k2":true}`,
			`{"type": "object", "properties": {"k1": {"type": "boolean"}}}`},
		{`{"k1":"new1","k2":"new2"}`, `{"k1":"old1"}`, `{"k1":"new1","k2":"new2"}`,
			`{"type": "object", "properties": {"k1": {"type": "string"}}}`},
		{`{"k1":123.6,"k2":123.9}`, `{"k1":234.3}`, `{"k1":123.6,"k2":123.9}`,
			`{"type": "object", "properties": {"k1": {"type": "number"}}}`},
		{`{"k1":123,"k2":345}`, `{"k1":234}`, `{"k1":123,"k2":345}`,
			`{"type": "object", "properties": {"k1": {"type": "integer"}}}`},
		{`{"k1":null,"k2":null}`, `{"k1":null}`, `{"k1":null,"k2":null}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}}`},
		{`{"k1":true,"k2":true}`, `{"k1":false}`, `{"k1":true}`,
			`{"type": "object", "properties": {"k1": {"type": "boolean"}}, "additionalProperties": false}`},
		{`{"k1":"new1","k2":"new2"}`, `{"k1":"old1"}`, `{"k1":"new1"}`,
			`{"type": "object", "properties": {"k1": {"type": "string"}}, "additionalProperties": false}`},
		{`{"k1":123.6,"k2":123.9}`, `{"k1":234.3}`, `{"k1":123.6}`,
			`{"type": "object", "properties": {"k1": {"type": "number"}}, "additionalProperties": false}`},
		{`{"k1":123,"k2":345}`, `{"k1":234}`, `{"k1":123}`,
			`{"type": "object", "properties": {"k1": {"type": "integer"}}, "additionalProperties": false}`},
		{`{"k1":null,"k2":null}`, `{"k1":null}`, `{"k1":null}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}, "additionalProperties": false}`},
		{`{"k1":true,"k2":true}`, `{"k1":false,"k3":false}`, `{"k1":true,"k2":true}`,
			`{"type": "object", "properties": {"k1": {"type": "boolean"}}, "additionalProperties": true}`},
		{`{"k1":"new1","k2":"new2"}`, `{"k1":"old1","k3":"old3"}`, `{"k1":"new1","k2":"new2"}`,
			`{"type": "object", "properties": {"k1": {"type": "string"}}, "additionalProperties": true}`},
		{`{"k1":123.6,"k2":123.9}`, `{"k1":234.3,"k3":234.6}`, `{"k1":123.6,"k2":123.9}`,
			`{"type": "object", "properties": {"k1": {"type": "number"}}, "additionalProperties": true}`},
		{`{"k1":123,"k2":345}`, `{"k1":234,"k3":456}`, `{"k1":123,"k2":345}`,
			`{"type": "object", "properties": {"k1": {"type": "integer"}}, "additionalProperties": true}`},
		{`{"k1":null,"k2":null}`, `{"k1":null,"k3":null}`, `{"k1":null,"k2":null}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}, "additionalProperties": true}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaAllOf(t *testing.T) {
	schemas := []string{
		`{
			"type": "object",
			"properties": {
				"k": {
					"allOf": [
						{"type": "array", "items": {"type": "string"}, "maxItems": 2},
						{"type": "array", "items": {"type": "string"}, "minItems": 2}
					]
				}
			}
		}`,
		`{
			"allOf": [
				{"type": "object", "properties": {"k1": {"type": "string"}}, "required": ["k1"]},
				{"type": "object", "properties": {"k2": {"type": "string"}}, "required": ["k2"]}
			]
		}`,
	}
	tests := []jsonSchemaTest{
		{`{"k":["new1","new2"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","new2"]}`, schemas[0]},
		{`{"k":["new1","new2","new3"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","new2"]}`, schemas[0]},
		{`{"k":["new1"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","old2"]}`, schemas[0]},
		{`{"k":[]}`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`{"k":"new"}`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`"new"`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`{}`, `{"k":["old1","old2"]}`, `{}`, schemas[0]},
		{`{"k1":"new1","k2":234}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1","k2":"old2"}`, schemas[1]},
		{`{"k1":123,"k2":"new2"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"new2"}`, schemas[1]},
		{`{"k1":"new1"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1","k2":"old2"}`, schemas[1]},
		{`{"k2":"new2"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"new2"}`, schemas[1]},
		{`"new"`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"old2"}`, schemas[1]},
		{`{}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"old2"}`, schemas[1]},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaAnyOfOrOneOf(t *testing.T) {
	schemas := []string{
		`{
			"type": "object",
			"properties": {
				"k": {
					"anyOf": [
						{"type": "array", "items": {"type": "string"}, "maxItems": 2},
						{"type": "array", "items": {"type": "string"}, "minItems": 2}
					]
				}
			}
		}`,
		`{
			"anyOf": [
				{"type": "object", "properties": {"k1": {"type": "string"}}, "required": ["k1"]},
				{"type": "object", "properties": {"k2": {"type": "string"}}, "required": ["k2"]}
			]
		}`,
		`{
			"anyOf": [
				{"type": "array", "items": {"type": "string", "enum": ["new1", "old2"]}},
				{"type": "array", "items": {"type": "string", "enum": ["old1", "old2"]}}
			]
		}`,
		`{
			"oneOf": [
				{"type": "array", "items": {"type": "string", "enum": ["new1", "old2"]}},
				{"type": "array", "items": {"type": "string", "enum": ["old1", "old2"]}}
			]
		}`,
	}
	tests := []jsonSchemaTest{
		{`{"k":["new1","new2"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","new2"]}`, schemas[0]},
		{`{"k":["new1","new2","new3"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","new2","new3"]}`, schemas[0]},
		{`{"k":["new1"]}`, `{"k":["old1","old2"]}`, `{"k":["new1"]}`, schemas[0]},
		{`{"k":[]}`, `{"k":["old1","old2"]}`, `{"k":[]}`, schemas[0]},
		{`{"k":"new"}`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`"new"`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`{}`, `{"k":["old1","old2"]}`, `{}`, schemas[0]},
		{`{"k1":"new1","k2":234}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1","k2":234}`, schemas[1]},
		{`{"k1":123,"k2":"new2"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":123,"k2":"new2"}`, schemas[1]},
		{`{"k1":"new1"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1"}`, schemas[1]},
		{`{"k2":"new2"}`, `{"k1":"old1","k2":"old2"}`, `{"k2":"new2"}`, schemas[1]},
		{`"new"`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"old2"}`, schemas[1]},
		{`{}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1"}`, schemas[1]},
		{`["new1","new2"]`, `["old1","old2"]`, `["new1","old2"]`, schemas[2]},
		{`["new1","new2","new3"]`, `["old1","old2"]`, `["new1","old2"]`, schemas[2]},
		{`["new1"]`, `["old1","old2"]`, `["new1"]`, schemas[2]},
		{`[]`, `["old1","old2"]`, `[]`, schemas[2]},
		{`["new1","new2"]`, `["old1","old2"]`, `["new1","old2"]`, schemas[3]},
		{`["new1","new2","new3"]`, `["old1","old2"]`, `["new1","old2"]`, schemas[3]},
		{`["new1"]`, `["old1","old2"]`, `["new1"]`, schemas[3]},
		{`[]`, `["old1","old2"]`, `["old1","old2"]`, schemas[3]},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaInvalidSchema(t *testing.T) {
	tests := []jsonSchemaTest{
		{`[[true], [true]]`, `[[false]]`, `[[false]]`, `{"type": "object"}`},
		{`[["new1"], ["new2"]]`, `[["old"]]`, `[["old"]]`, `{"type": "object"}`},
		{`[[123.6], [123.9]]`, `[[234.3]]`, `[[234.3]]`, `{"type": "object"}`},
		{`[[123], [234]]`, `[[234]]`, `[[234]]`, `{"type": "object"}`},
		{`[[null], [null]]`, `[[null]]`, `[[null]]`, `{"type": "object"}`},
		{`{"k1":true,"k2":true}`, `{"k1":false,"k3":false}`, `{"k2":true}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}}`},
		{`{"k1":"new1","k2":"new2"}`, `{"k1":"old1","k3":"old3"}`, `{"k2":"new2"}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}}`},
		{`{"k1":123.6,"k2":123.9}`, `{"k1":234.3,"k3":234.6}`, `{"k2":123.9}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}}`},
		{`{"k1":123,"k2":345}`, `{"k1":234,"k3":456}`, `{"k2":345}`,
			`{"type": "object", "properties": {"k1": {"type": "null"}}}`},
		{`{"k1":true,"k2":true}`, `{"k1":false,"k3":false}`, `{"k1":true,"k2":true,"k3":false}`,
			`{"type": "object", "required": ["k3"]}`},
		{`{"k1":"new1","k2":"new2"}`, `{"k1":"old1","k3":"old3"}`, `{"k1":"new1","k2":"new2","k3":"old3"}`,
			`{"type": "object", "required": ["k3"]}`},
		{`{"k1":123.6,"k2":123.9}`, `{"k1":234.3,"k3":234.6}`, `{"k1":123.6,"k2":123.9,"k3":234.6}`,
			`{"type": "object", "required": ["k3"]}`},
		{`{"k1":123,"k2":345}`, `{"k1":234,"k3":456}`, `{"k1":123,"k2":345,"k3":456}`,
			`{"type": "object", "required": ["k3"]}`},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaRef(t *testing.T) {
	schemas := []string{
		`{
			"definitions": {
				"r.k": {
					"allOf": [
						{"type": "array", "items": {"type": "string"}, "maxItems": 2},
						{"type": "array", "items": {"type": "string"}, "minItems": 2}
					]
				}
			},
			"type": "object",
			"properties": {
				"k": {
					"$ref": "#/definitions/r.k"
				}
			}
		}`,
		`{
			"definitions": {
				"r": {
					"allOf": [
						{"type": "object", "properties": {"k1": {"type": "string"}}, "required": ["k1"]},
						{"type": "object", "properties": {"k2": {"type": "string"}}, "required": ["k2"]}
					]
				}
			},
			"$ref": "#/definitions/r"
		}`,
	}
	tests := []jsonSchemaTest{
		{`{"k":["new1","new2"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","new2"]}`, schemas[0]},
		{`{"k":["new1","new2","new3"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","new2"]}`, schemas[0]},
		{`{"k":["new1"]}`, `{"k":["old1","old2"]}`, `{"k":["new1","old2"]}`, schemas[0]},
		{`{"k":[]}`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`{"k":"new"}`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`"new"`, `{"k":["old1","old2"]}`, `{"k":["old1","old2"]}`, schemas[0]},
		{`{}`, `{"k":["old1","old2"]}`, `{}`, schemas[0]},
		{`{"k1":"new1","k2":234}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1","k2":"old2"}`, schemas[1]},
		{`{"k1":123,"k2":"new2"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"new2"}`, schemas[1]},
		{`{"k1":"new1"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"new1","k2":"old2"}`, schemas[1]},
		{`{"k2":"new2"}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"new2"}`, schemas[1]},
		{`"new"`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"old2"}`, schemas[1]},
		{`{}`, `{"k1":"old1","k2":"old2"}`, `{"k1":"old1","k2":"old2"}`, schemas[1]},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}

func TestMergeTwoJSONsBySchemaSecureConfig(t *testing.T) {
	schemaTmpl := `{
           "type": "object",
           "properties": {
             "k": {
               "type": "object",
               "required": [
                 "server_only",
                 "value"
               ],
               "properties": {
                 "value": {
                   %s
                 },
                 "server_only": {
                   "enum": [
                     true
                   ],
                   "type": "boolean",
                   "value": true
                 }
               }
             }
           }
		}`
	var (
		schemaString     = fmt.Sprintf(schemaTmpl, `"type": "string"`)
		schemaInt        = fmt.Sprintf(schemaTmpl, `"type": "integer"`)
		schemaNum        = fmt.Sprintf(schemaTmpl, `"type": "number"`)
		schemaBool       = fmt.Sprintf(schemaTmpl, `"type": "boolean"`)
		schemaNull       = fmt.Sprintf(schemaTmpl, `"type": "null"`)
		schemaArrString  = fmt.Sprintf(schemaTmpl, `"type": "array", "items": {"type": "string"}`)
		schemaArrInt     = fmt.Sprintf(schemaTmpl, `"type": "array", "items": {"type": "integer"}`)
		schemaArrNum     = fmt.Sprintf(schemaTmpl, `"type": "array", "items": {"type": "number"}`)
		schemaArrBool    = fmt.Sprintf(schemaTmpl, `"type": "array", "items": {"type": "boolean"}`)
		schemaObjSimple  = fmt.Sprintf(schemaTmpl, `"type": "object", "properties": {"k1": {"type":"string"}, "k2": {"type":"integer"}}`)
		schemaObjComplex = fmt.Sprintf(schemaTmpl, `"type": "object", "properties": {"k1": {"type":"string"}, "k2": {"type":"object", "properties": {"k3": {"type":"number"}}}}`)
		schemaObjArr     = fmt.Sprintf(schemaTmpl, `"type": "object", "properties": {"k1": {"type":"array"}, "items": {"type": "integer"}}`)
	)

	tests := []jsonSchemaTest{
		{`{"k":{"value": "new","server_only": true}}`, `{"k":{"value": "old","server_only": true}}`, `{"k":{"value": "new","server_only": true}}`, schemaString},
		{`{"k":{"value": 123,"server_only": true}}`, `{"k":{"value": 678,"server_only": true}}`, `{"k":{"value": 123,"server_only": true}}`, schemaInt},
		{`{"k":{"value": 123.1,"server_only": true}}`, `{"k":{"value": 678.8,"server_only": true}}`, `{"k":{"value": 123.1,"server_only": true}}`, schemaNum},
		{`{"k":{"value": true,"server_only": true}}`, `{"k":{"value": false,"server_only": true}}`, `{"k":{"value": true,"server_only": true}}`, schemaBool},
		{`{"k":{"value": false,"server_only": true}}`, `{"k":{"value": null,"server_only": true}}`, `{"k":{"value": null,"server_only": true}}`, schemaNull},
		{`{"k":{"value": ["s1", "s2"],"server_only": true}}`, `{"k":{"value": ["s3"],"server_only": true}}`, `{"k":{"value": ["s1", "s2"],"server_only": true}}`, schemaArrString},
		{`{"k":{"value": [11, 22],"server_only": true}}`, `{"k":{"value": [33],"server_only": true}}`, `{"k":{"value": [11, 22],"server_only": true}}`, schemaArrInt},
		{`{"k":{"value": [1.1, 2.2],"server_only": true}}`, `{"k":{"value": [3.3],"server_only": true}}`, `{"k":{"value": [1.1, 2.2],"server_only": true}}`, schemaArrNum},
		{`{"k":{"value": [true, false],"server_only": true}}`, `{"k":{"value": [false],"server_only": true}}`, `{"k":{"value": [true, false],"server_only": true}}`, schemaArrBool},
		{`{"k":{"value": {"k1":"v1", "k2":1},"server_only": true}}`, `{"k":{"value": {"k1":"v2"},"server_only": true}}`, `{"k":{"value": {"k1":"v1", "k2":1},"server_only": true}}`, schemaObjSimple},
		{`{"k":{"value": {"k1":"v1", "k2":{"k3":1.2}},"server_only": true}}`, `{"k":{"value": {"k1":"v2"},"server_only": true}}`, `{"k":{"value": {"k1":"v1", "k2":{"k3":1.2}},"server_only": true}}`, schemaObjComplex},
		{`{"k":{"value": {"k1":[11, 12]},"server_only": true}}`, `{"k":{"value": {"k1":[]},"server_only": true}}`, `{"k":{"value": {"k1":[11, 12]},"server_only": true}}`, schemaObjArr},
	}
	for _, test := range tests {
		makeTestCompareBySchema(t, test)
	}
}
