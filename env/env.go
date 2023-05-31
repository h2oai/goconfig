package env

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/h2oai/goconfig"
	"github.com/joho/godotenv"
)

func init() {
	goconfig.Formats = append(goconfig.Formats, goconfig.Fileformat{
		Extension:   ".env",
		Load:        LoadEnv,
		PrepareHelp: PrepareHelp,
	})
}

func LoadEnv(config interface{}) error {
	configFile := filepath.Join(goconfig.Path, goconfig.File)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// If config does not exist, just continue.
		return nil
	}

	dotEnvMap, err := godotenv.Read(configFile)
	if err != nil {
		return err
	}

	// Format .env file keys.
	for k, v := range dotEnvMap {
		delete(dotEnvMap, k)
		if strings.HasPrefix(k, goconfig.PrefixEnv) {
			k = strings.TrimPrefix(k, goconfig.PrefixEnv+"_")
		}
		dotEnvMap[strings.ToLower(k)] = v
	}

	configType := reflect.TypeOf(config).Elem()
	configValue := reflect.ValueOf(config).Elem()

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		tag := field.Tag.Get("cfg")
		if tag == "" {
			continue
		}
		value, ok := dotEnvMap[tag]
		if !ok {
			continue
		}
		switch field.Type.Kind() {
		case reflect.String:
			configValue.Field(i).SetString(value)
		case reflect.Int:
			intValue, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("failed to parse int value for field %s: %v", field.Name, err)
			}
			configValue.Field(i).SetInt(int64(intValue))
		case reflect.Bool:
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse bool value for field %s: %v", field.Name, err)
			}
			configValue.Field(i).SetBool(boolValue)
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.String {
				configValue.Field(i).Set(reflect.ValueOf(strings.Split(value, " ")))
				break
			}
			return fmt.Errorf("unsupported slice element type: %v", field.Type.Elem().Kind())
		default:
			return fmt.Errorf("unsupported field type: %v", field.Type.Kind())
		}
	}

	return nil
}

func PrepareHelp(config interface{}) (string, error) {
	var helpAux []byte
	configValue := reflect.ValueOf(config).Elem()
	for i := 0; i < configValue.NumField(); i++ {
		fieldName := configValue.Type().Field(i).Name
		var snakeCase []byte
		for i, c := range fieldName {
			if i > 0 && c >= 'A' && c <= 'Z' {
				snakeCase = append(snakeCase, '_')
			}
			snakeCase = append(snakeCase, byte(c))
		}
		prefix := ""
		if goconfig.PrefixEnv != "" {
			prefix = goconfig.PrefixEnv + "_"
		}
		helpAux = append(helpAux, []byte(prefix+strings.ToUpper(string(snakeCase)))...)
		helpAux = append(helpAux, []byte("=value\n")...)
	}
	return string(helpAux), nil
}
