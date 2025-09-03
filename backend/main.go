package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"winmanager-backend/internal/config"
	"winmanager-backend/internal/controllers"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"
	"winmanager-backend/internal/services"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	name        = "winmanager-backend"
	description = "WinManager Backend Server"
	version     = "1.0.0"
)

func init() {
	// 先设置基本的日志输出到控制台
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// 初始化配置
	config.Init()

	// 初始化日志系统
	logger.Init()

	logger.Infof("启动 %s 版本 %s", name, version)

	// 初始化数据库
	models.Init()

	// 初始化离线检测服务
	services.InitOfflineDetector()
}

func customVersionPrinter(c *cli.Context) {
	fmt.Println(version)
}

func main() {
	cli.VersionPrinter = customVersionPrinter
	app := &cli.App{
		Name:        name,
		Usage:       description,
		Version:     version,
		Description: description,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "http",
				Aliases: []string{"a"},
				Value:   ":8080",
				Usage:   "HTTP监听地址",
			},
		},
		Action: run,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// 创建Gin应用
	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.Use(gin.Recovery())
	
	// 设置路由
	controllers.InitRouter(app.Group("/api"))
	
	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    c.String("http"),
		Handler: app,
	}

	// 启动服务器
	go func() {
		logger.Infof("HTTP服务器启动在 %s", c.String("http"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Warnf("服务器正在关闭...")

	// 停止离线检测服务
	services.StopOfflineDetector()

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}
	<-ctx.Done()
	logger.Infof("服务器已退出")
	return nil
}
