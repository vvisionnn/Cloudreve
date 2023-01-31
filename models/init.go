package model

import (
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v3/pkg/conf"
	"github.com/cloudreve/Cloudreve/v3/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// DB 数据库链接单例
var DB *gorm.DB

// Init 初始化 MySQL 链接
func Init() {
	util.Log().Info("Initializing database connection...")

	var (
		db  *gorm.DB
		err error
	)

	if gin.Mode() == gin.TestMode {
		// 测试模式下，使用内存数据库
		db, err = gorm.Open("sqlite3", ":memory:")
	} else {
		switch conf.DatabaseConfig.Type {
		case "UNSET", "sqlite", "sqlite3":
			// 未指定数据库或者明确指定为 sqlite 时，使用 SQLite3 数据库
			db, err = gorm.Open("sqlite3", util.RelativePath(conf.DatabaseConfig.DBFile))
		case "postgres":
			db, err = gorm.Open(conf.DatabaseConfig.Type, fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
				conf.DatabaseConfig.Host,
				conf.DatabaseConfig.User,
				conf.DatabaseConfig.Password,
				conf.DatabaseConfig.Name,
				conf.DatabaseConfig.Port))
		case "mysql", "mssql":
			var host string
			if conf.DatabaseConfig.UnixSocket {
				host = fmt.Sprintf("unix(%s)",
					conf.DatabaseConfig.Host)
			} else {
				host = fmt.Sprintf("(%s:%d)",
					conf.DatabaseConfig.Host,
					conf.DatabaseConfig.Port)
			}

			db, err = gorm.Open(conf.DatabaseConfig.Type, fmt.Sprintf("%s:%s@%s/%s?charset=%s&parseTime=True&loc=Local",
				conf.DatabaseConfig.User,
				conf.DatabaseConfig.Password,
				host,
				conf.DatabaseConfig.Name,
				conf.DatabaseConfig.Charset))
		default:
			util.Log().Panic("Unsupported database type %q.", conf.DatabaseConfig.Type)
		}
	}

	//db.SetLogger(util.Log())
	if err != nil {
		util.Log().Panic("Failed to connect to database: %s", err)
	}

	// 处理表前缀
	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return conf.DatabaseConfig.TablePrefix + defaultTableName
	}

	// Debug模式下，输出所有 SQL 日志
	if conf.SystemConfig.Debug {
		db.LogMode(true)
	} else {
		db.LogMode(false)
	}

	//设置连接池
	db.DB().SetMaxIdleConns(50)
	if conf.DatabaseConfig.Type == "sqlite" || conf.DatabaseConfig.Type == "sqlite3" || conf.DatabaseConfig.Type == "UNSET" {
		db.DB().SetMaxOpenConns(1)
	} else {
		db.DB().SetMaxOpenConns(100)
	}

	//超时
	db.DB().SetConnMaxLifetime(time.Second * 30)

	DB = db

	//执行迁移
	migration()
}
