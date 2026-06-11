package client

import (
	"SREFlow/config"
	"SREFlow/internal/models"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/zeromicro/go-zero/core/logc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDBClient(dbcfg config.Database) *gorm.DB {
	var db *gorm.DB
	var err error

	// 设置默认数据库类型为 mysql
	if dbcfg.Type == "" {
		dbcfg.Type = "mysql"
	}

	switch dbcfg.Type {
	case "sqlite":
		db, err = initSQLiteDB(dbcfg)
	case "mysql":
		db, err = initMySQLDB(dbcfg)
	default:
		logc.Errorf(context.Background(), "unsupported database type: %s", dbcfg.Type)
		return nil
	}

	if err != nil {
		logc.Errorf(context.Background(), "failed to connect database: %s", err.Error())
		return nil
	}

	err = db.AutoMigrate(
		&models.UserModel{},
		&models.Workflow{},
		&models.WorkflowExecutionRecord{},
		&models.Integration{},
		&models.Workspace{},
		&models.WorkflowVersion{},
		&models.MCPServer{},
		&models.Skill{},
	)
	if err != nil {
		logc.Error(context.Background(), err.Error())
		return nil
	}

	if config.Application.Server.Mode == "debug" {
		db.Debug()
	} else {
		db.Logger = logger.Default.LogMode(logger.Silent)
	}

	return db
}

// initMySQLDB 初始化 MySQL 数据库连接
func initMySQLDB(config config.Database) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local&timeout=%s",
		config.User,
		config.Pass,
		config.Host,
		config.Port,
		config.DBName,
		config.Timeout)

	logc.Infof(context.Background(), "connecting to MySQL database: %s:%s/%s", config.Host, config.Port, config.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// initSQLiteDB 初始化 SQLite 数据库连接
func initSQLiteDB(config config.Database) (*gorm.DB, error) {
	// 设置默认 SQLite 文件路径
	if config.Path == "" {
		config.Path = "data/watchalert.db"
	}

	// 确保目录存在
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	logc.Infof(context.Background(), "connecting to SQLite database: %s", config.Path)
	return gorm.Open(sqlite.Open(config.Path), &gorm.Config{})
}
