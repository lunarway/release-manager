package regexp

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Compile takes a regular expression and a custom lookup struct that should match the named groups
// in the regexp.
//
// `lookup` should always be a pointer to a struct with only exported `int` properties.
func Compile(re string, lookup interface{}) (*regexp.Regexp, error) {
	var regexpCompiled = regexp.MustCompile(re)

	lookupValue := reflect.ValueOf(lookup)
	if lookupValue.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("lookup must be a pointer")
	}

	lookupElem := lookupValue.Elem()
	subexpNames := map[string]int{}
	for i, name := range regexpCompiled.SubexpNames() {
		if name == "" {
			continue
		}
		if cases.Title(language.English, cases.NoLower).String(name) != name {
			return nil, fmt.Errorf("field '%s' in regexp `%s` is not capitalized", name, re)
		}
		subexpNames[name] = i
	}

	for i := 0; i < lookupElem.NumField(); i++ {
		field := lookupElem.Type().Field(i)
		fieldName := field.Name
		index, foundIndex := subexpNames[fieldName]
		if !foundIndex {
			return nil, fmt.Errorf("field '%s' is not in regexp `%s`", fieldName, re)
		}
		lookupElem.FieldByName(fieldName).SetInt(int64(index))
		delete(subexpNames, fieldName)
	}

	var unusedNames []string
	for name := range subexpNames {
		unusedNames = append(unusedNames, name)
	}

	if len(unusedNames) > 0 {
		return nil, fmt.Errorf("regexp `%s` has named groups not found in lookup: %s", re, strings.Join(unusedNames, ", "))
	}

	return regexpCompiled, nil
}

func MustCompile(re string, lookup interface{}) *regexp.Regexp {
	r, err := Compile(re, lookup)
	if err != nil {
		panic(err)
	}
	return r
}
