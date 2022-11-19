package database

import (
	"context"
	"fmt"
	"os"
	"rsshub-auto-refresh/component/logger"
	"rsshub-auto-refresh/config"

	"github.com/gogf/gf/v2/encoding/gjson"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	dbInstance *gorm.DB
)

func InitDatabase(ctx context.Context) {
	databaseType := config.GetConfig().Get("database.type").String()
	logger.Log().Infof(ctx, "database type is : %s \n", databaseType)
	setMySQLConfig(ctx)
}

func setSQLiteConfig(ctx context.Context) {
	var (
		err      error
		dbConfig gorm.Config
    dbPath string
	)
  dbPath = config.GetConfig().Get("database.sqlite.dbPath").String()
	if os.Getenv("env") == "dev" {
		dbConfig = gorm.Config{
			Logger: gormLogger.Default.LogMode(gormLogger.Info),
		}
	} else {
		dbConfig = gorm.Config{
			Logger: gormLogger.Default.LogMode(gormLogger.Error),
		}
	}

	dbInstance, err = gorm.Open(sqlite.Open(dbPath), &dbConfig)
	if err != nil {
		logger.Log().Error(ctx, err)
	}
}

func setMySQLConfig(ctx context.Context) {
	var (
		err         error
		dbConfig    gorm.Config
		mysqlConfig *gjson.Json
		dsn         string
		user        string
		password    string
		url         string
		dbName      string
	)

	mysqlConfig = config.GetConfig().GetJson("database.mysql")
	user = mysqlConfig.Get("user").String()
	password = mysqlConfig.Get("password").String()
	url = mysqlConfig.Get("url").String()
	dbName = mysqlConfig.Get("dbName").String()

	dsn = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, url, dbName)
	dbInstance, err = gorm.Open(mysql.Open(dsn), &dbConfig)
	if err != nil {
		logger.Log().Error(ctx, err)
		panic(err)
	}
}

func GetDatabase() *gorm.DB {
	return dbInstance
}
