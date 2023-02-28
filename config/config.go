package config

import (
	"embed"
	"os"

	"github.com/gogf/gf/v2/encoding/gjson"
)

//go:embed *
var ConfigFS embed.FS

var ConfigJson *gjson.Json

func GetConfig() *gjson.Json {
	var (
		env       string
		configStr []byte
		err       error
	)
	if !ConfigJson.IsNil() {
		return ConfigJson
	}
	env = os.Getenv("env")
	if env == "dev" {
		configStr, err = ConfigFS.ReadFile("config.dev.json")
	} else {
		configStr, err = ConfigFS.ReadFile("config.json")
	}

	if err != nil {
		panic(err)
	}

	ConfigJson = gjson.New(configStr)
	return ConfigJson
}
