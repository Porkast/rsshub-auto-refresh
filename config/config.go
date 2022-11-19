package config

import (
	"embed"

	"github.com/gogf/gf/v2/encoding/gjson"
)

//go:embed *
var ConfigFS embed.FS

func GetConfig() *gjson.Json {
	configStr, err := ConfigFS.ReadFile("config.json")
	if err != nil {
		panic(err)
	}

	configJson := gjson.New(configStr)
	return configJson
}
