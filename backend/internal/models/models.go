package models

import (
	"fmt"
	"os"
	"winmanager-backend/internal/config"
	"winmanager-backend/internal/logger"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// DB 全局数据库连接
var DB *gorm.DB

// Init 初始化数据库连接
func Init() {
	var err error

	// 获取数据库路径
	dbPath := config.GetDatabasePath()

	// 检查数据库文件是否存在
	isFirstRun := !fileExists(dbPath)
	if isFirstRun {
		logger.Infof("首次运行，将创建新的数据库文件: %s", dbPath)
	}

	// 连接SQLite数据库
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	logger.Infof("数据库连接成功: %s", dbPath)

	// 自动迁移数据表
	if err := autoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 如果是首次运行，创建一些初始数据
	if isFirstRun {
		if err := createInitialData(); err != nil {
			logger.Warnf("创建初始数据失败: %v", err)
		}
	}

	logger.Infof("数据库初始化完成")
}

// autoMigrate 自动迁移数据表
func autoMigrate() error {
	// 迁移实例表
	if err := DB.AutoMigrate(&Instance{}); err != nil {
		return fmt.Errorf("迁移实例表失败: %v", err)
	}

	// 迁移分组表
	if err := DB.AutoMigrate(&Group{}); err != nil {
		return fmt.Errorf("迁移分组表失败: %v", err)
	}

	logger.Infof("数据表迁移完成")
	return nil
}

// fileExists 检查文件是否存在
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// createInitialData 创建初始数据
func createInitialData() error {
	// 创建默认分组
	defaultGroup := Group{
		Name:  "默认分组",
		Total: 0,
	}

	if err := DB.Create(&defaultGroup).Error; err != nil {
		return fmt.Errorf("创建默认分组失败: %v", err)
	}

	logger.Infof("创建初始数据完成")
	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}

// Close 关闭数据库连接
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
