package logger

import (
	"rsshub-auto-refresh/config"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"
)

var logger *glog.Logger

func Init() {
	logger = glog.New()
	configJson := config.GetConfig()
	logger.SetConfigWithMap(g.Map{
		"path":  configJson.Get("logPath").String(),
		"level": "all",
		"RotateSize": "100M",
		"RotateBackupLimit": 2,
		"RotateBackupExpire": "3d",
		"RotateBackupCompress": 9,
	})
}

func Log() *glog.Logger {
	return logger
}
