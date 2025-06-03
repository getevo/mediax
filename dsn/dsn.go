package dsn

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ParseDSN(dsn string, config any) error {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("config must be a non-nil pointer")
	}
	val = val.Elem()
	typ := val.Type()

	// Extract a DSN pattern
	var dsnPattern string
	for i := 0; i < typ.NumField(); i++ {
		if tag := typ.Field(i).Tag.Get("dsn"); tag != "" {
			dsnPattern = tag
			break
		}
	}
	if dsnPattern == "" {
		return fmt.Errorf("missing `dsn` tag in any field")
	}
	var re = regexp.MustCompile(`(?mi)(?P<scheme>[a-z0-9()]+):[\\/]{2}(\$(?P<username>[a-z0-9]+):\$(?P<password>[a-z0-9]+)@)?\$(?P<host>[a-z0-9]+)((:\$(?P<port>[a-z0-9]+))?[\\/]\$(?P<path>[a-z0-9]+))?`)

	matches := FindAllGroups(re, dsnPattern)

	if len(matches) == 0 {
		return fmt.Errorf("failed to extract DSN")
	}
	dsnPattern = strings.ReplaceAll(dsnPattern, `/`, `\/`)
	for key, value := range matches {
		if value == "" {
			continue
		}
		switch key {
		case "scheme":
			dsnPattern = strings.Replace(dsnPattern, value, `(?P<scheme>`+AddOptionalToGroups(value)+`)`, 1)
		case "port":
			dsnPattern = strings.Replace(dsnPattern, "$"+value, `(?P<`+value+`>[\w]*)`, 1)
		case "username":
			dsnPattern = strings.Replace(dsnPattern, "$"+value, `(?P<`+value+`>[\w\\\/@\-\+#!$%^&*()=\."]*)`, 1)
		case "host", "path", "password":
			dsnPattern = strings.Replace(dsnPattern, "$"+value, `(?P<`+value+`>[\w\\\/@\-\+#!$%^&*()=:\."]*)`, 1)
		}
	}
	re = regexp.MustCompile(`(?mi)` + dsnPattern)
	schemeRegex := regexp.MustCompile(`(?P<scheme>[a-z0-9()]+):[\\/]{2}`)
	groups := FindAllGroups(schemeRegex, dsn)
	var query string
	dsn, query = SplitLast(dsn, "?")
	matches = FindAllGroups(re, dsn)
	if len(matches) == 0 {
		return fmt.Errorf("failed to extract DSN")
	}
	if v, ok := groups["scheme"]; !ok {
		return fmt.Errorf("failed to extract scheme")
	} else {
		matches["DSN"] = dsn
		matches["Scheme"] = v
	}
	queryString, err := parseQueryString(query)
	if err != nil {
		return err
	}
	for k, v := range queryString {
		matches[k] = v
	}
	for i := 0; i < typ.NumField(); i++ {
		var fieldName = typ.Field(i).Name
		var field = val.Field(i)
		if fieldName == "Params" {
			field.Set(reflect.ValueOf(queryString))
			continue
		}
		if v, ok := matches[fieldName]; ok {
			setField(field, v)
		} else if tag := typ.Field(i).Tag.Get("default"); tag != "" {
			setField(field, tag)
		}
	}
	return nil
}

func setField(field reflect.Value, val string) {
	if !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(val)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t := field.Type(); t.PkgPath() == "time" && t.Name() == "Duration" {
			if d, err := time.ParseDuration(val); err == nil {
				field.SetInt(int64(d))
				return
			}
		}
		if i, err := strconv.ParseInt(val, 10, field.Type().Bits()); err == nil {
			field.SetInt(i)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if u, err := strconv.ParseUint(val, 10, field.Type().Bits()); err == nil {
			field.SetUint(u)
		}

	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(val, field.Type().Bits()); err == nil {
			field.SetFloat(f)
		}

	case reflect.Bool:
		if b, err := strconv.ParseBool(val); err == nil {
			field.SetBool(b)
		}

	case reflect.Slice:
		elemKind := field.Type().Elem().Kind()
		if elemKind == reflect.String {
			parts := strings.Split(val, ",")
			slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
			for i, p := range parts {
				slice.Index(i).SetString(strings.TrimSpace(p))
			}
			field.Set(slice)
		}
	default:
		return
	}
}

func FindAllGroups(re *regexp.Regexp, s string) map[string]string {
	matches := re.FindStringSubmatch(s)
	subnames := re.SubexpNames()
	if matches == nil || len(matches) != len(subnames) {
		return nil
	}

	matchMap := map[string]string{}
	for i := 1; i < len(matches); i++ {
		matchMap[subnames[i]] = matches[i]
	}
	return matchMap
}

func SplitLast(s, sep string) (string, string) {
	idx := strings.LastIndex(s, sep)
	if idx == -1 {
		return s, "" // separator not found
	}
	return s[:idx], s[idx+len(sep):]
}

func parseQueryString(query string) (map[string]string, error) {
	result := make(map[string]string)

	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	for key, val := range values {
		if len(val) > 0 {
			result[key] = val[0] // Take only the first value
		}
	}

	return result, nil
}

func AddOptionalToGroups(input string) string {
	var result strings.Builder
	inGroup := false

	for _, r := range input {
		switch r {
		case '(':
			inGroup = true
			result.WriteRune(r)
		case ')':
			if inGroup {
				result.WriteRune(r)
				result.WriteRune('?') // make group optional
				inGroup = false
			} else {
				result.WriteRune(r) // unbalanced ')' — still write
			}
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}
