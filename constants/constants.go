// Package constants  contains all the constants used by thorchain
// by default all the settings in this is for mainnet
package constants

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blang/semver"
)

var (
	GitCommit       string = "null"  // sha1 revision used to build the program
	BuildTime       string = "null"  // when the executable was built
	Version         string = "0.1.0" // software version
	int64Overrides         = map[ConstantName]int64{}
	boolOverrides          = map[ConstantName]bool{}
	stringOverrides        = map[ConstantName]string{}
)

// The version of this software
var SWVersion, _ = semver.Make(Version)

// Block time of THORChain
var ThorchainBlockTime = 5 * time.Second

// ConstantValues implement ConstantValues interface
type ConstantVals struct {
	int64values  map[ConstantName]int64
	boolValues   map[ConstantName]bool
	stringValues map[ConstantName]string
}

// GetInt64Value get value in int64 type, if it doesn't exist then it will return the default value of int64, which is 0
func (cv *ConstantVals) GetInt64Value(name ConstantName) int64 {
	// check overrides first
	v, ok := int64Overrides[name]
	if ok {
		return v
	}

	v, ok = cv.int64values[name]
	if !ok {
		return 0
	}
	return v
}

// GetBoolValue retrieve a bool constant value from the map
func (cv *ConstantVals) GetBoolValue(name ConstantName) bool {
	v, ok := boolOverrides[name]
	if ok {
		return v
	}
	v, ok = cv.boolValues[name]
	if !ok {
		return false
	}
	return v
}

// GetStringValue retrieve a string const value from the map
func (cv *ConstantVals) GetStringValue(name ConstantName) string {
	v, ok := stringOverrides[name]
	if ok {
		return v
	}
	v, ok = cv.stringValues[name]
	if ok {
		return v
	}
	return ""
}

func (cv *ConstantVals) String() string {
	sb := strings.Builder{}
	for k, v := range cv.int64values {
		if overrideValue, ok := int64Overrides[k]; ok {
			sb.WriteString(fmt.Sprintf("%s:%d\n", k, overrideValue))
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:%d\n", k, v))
	}
	for k, v := range cv.boolValues {
		if overrideValue, ok := boolOverrides[k]; ok {
			sb.WriteString(fmt.Sprintf("%s:%v\n", k, overrideValue))
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:%v\n", k, v))
	}
	return sb.String()
}

// MarshalJSON marshal result to json format
func (cv ConstantVals) MarshalJSON() ([]byte, error) {
	var result struct {
		Int64Values  map[string]int64  `json:"int_64_values"`
		BoolValues   map[string]bool   `json:"bool_values"`
		StringValues map[string]string `json:"string_values"`
	}
	result.Int64Values = make(map[string]int64)
	result.BoolValues = make(map[string]bool)
	result.StringValues = make(map[string]string)
	for k, v := range cv.int64values {
		result.Int64Values[k.String()] = v
	}
	for k, v := range int64Overrides {
		result.Int64Values[k.String()] = v
	}
	for k, v := range cv.boolValues {
		result.BoolValues[k.String()] = v
	}
	for k, v := range boolOverrides {
		result.BoolValues[k.String()] = v
	}
	for k, v := range cv.stringValues {
		result.StringValues[k.String()] = v
	}
	for k, v := range stringOverrides {
		result.StringValues[k.String()] = v
	}

	return json.MarshalIndent(result, "", "	")
}
