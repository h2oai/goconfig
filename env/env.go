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

	configType := reflect.TypeOf(config).Elem()
	configValue := reflect.ValueOf(config).Elem()

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		confKey := getConfKey(field)
		if confKey == "-" {
			continue
		}

		prefix := ""
		if goconfig.PrefixEnv != "" {
			prefix = goconfig.PrefixEnv + "_"
		}
		value, ok := dotEnvMap[prefix+confKey]
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
		confKey := getConfKey(configValue.Type().Field(i))
		if confKey == "-" {
			continue
		}

		prefix := ""
		if goconfig.PrefixEnv != "" {
			prefix = goconfig.PrefixEnv + "_"
		}
		helpAux = append(helpAux, []byte(prefix+strings.ToUpper(confKey)+"=value\n")...)
	}
	return string(helpAux), nil
}

func getConfKey(field reflect.StructField) string {
	k := field.Tag.Get("env")
	if k == "" {
		k = field.Tag.Get("cfg")
	}
	if k == "" {
		k = strings.ToUpper(field.Name)
	}
	return k
}
