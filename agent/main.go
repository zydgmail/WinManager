package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"winmanager-agent/internal/api"
	"winmanager-agent/internal/config"
	"winmanager-agent/internal/controllers"
	"winmanager-agent/internal/handlers"
	"winmanager-agent/internal/logger"
	pb "winmanager-agent/protos"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

const (
	appName        = "winmanager-agent"
	appDescription = "WinManager Agent - Remote Desktop Control Agent"
)

func main() {
	setupLogger()

	app := &cli.App{
		Name:        appName,
		Usage:       appDescription,
		Description: appDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.json",
				Usage:   "Configuration file path",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Value:   false,
				Usage:   "Enable debug mode",
			},
			&cli.StringFlag{
				Name:    "server",
				Aliases: []string{"s"},
				Value:   "",
				Usage:   "Server address for registration",
			},
			&cli.StringFlag{
				Name:    "grpc",
				Aliases: []string{"g"},
				Value:   "",
				Usage:   "gRPC server listen address",
			},
			&cli.StringFlag{
				Name:    "http",
				Aliases: []string{"a"},
				Value:   "",
				Usage:   "HTTP server listen address",
			},
		},
		Action: runAgent,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func setupLogger() {
	// 基础日志设置，避免在logger初始化前出现问题
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.InfoLevel)
}

func runAgent(c *cli.Context) error {
	// 使用全局配置实例，确保整个应用使用同一个配置
	cfg := config.GetGlobalConfig()

	// Load configuration file
	configPath := c.String("config")
	if err := cfg.LoadConfigFile(configPath); err != nil {
		log.WithError(err).Fatal("Failed to load configuration file")
	}

	// Setup logger based on debug flag (command line overrides config)
	debugMode := c.Bool("debug") || cfg.IsDebugMode()
	if debugMode {
		logger.SetupDebugLogger()
	} else {
		logger.SetupProductionLogger()
	}

	log.WithFields(log.Fields{
		"服务":   "WinManager Agent",
		"版本":   cfg.GetVersion(),
		"调试模式": debugMode,
		"配置文件": configPath,
	}).Info("正在启动 WinManager Agent")

	// Initialize configuration
	if err := cfg.Initialize(c.String("server")); err != nil {
		log.WithError(err).Fatal("Failed to initialize configuration")
	}

	// Register with server
	if err := registerWithServer(cfg); err != nil {
		log.WithError(err).Fatal("Failed to register with server")
	}

	// Initialize encoder service
	handlers.InitEncoderService()

	// Start metrics recording
	controllers.StartMetricsRecording()

	// Setup graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get server addresses (command line overrides config)
	grpcAddr := c.String("grpc")
	if grpcAddr == "" {
		grpcAddr = fmt.Sprintf(":%d", cfg.GetGRPCPort())
	}

	httpAddr := c.String("http")
	if httpAddr == "" {
		httpAddr = fmt.Sprintf(":%d", cfg.GetHTTPPort())
	}

	// Start gRPC server
	grpcServer := startGRPCServer(grpcAddr)
	defer grpcServer.GracefulStop()

	// Start HTTP server
	httpServer := startHTTPServer(httpAddr)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.WithError(err).Error("HTTP server shutdown error")
		}
	}()

	// Wait for shutdown signal
	waitForShutdown()

	log.Info("Agent 已完全关闭")
	return nil
}

func registerWithServer(cfg *config.Config) error {
	log.Info("Registering with server...")

	id, err := api.RegisterAgent(cfg.GetServerURL())
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	log.WithField("id", id).Info("Successfully registered with server")

	// Start heartbeat
	api.StartHeartbeat(cfg.GetServerURL())

	return nil
}

func startGRPCServer(address string) *grpc.Server {
	if address == "" {
		log.Info("gRPC server disabled")
		return nil
	}

	log.WithField("监听地址", address).Info("正在启动 gRPC 服务器")

	server := grpc.NewServer()
	pb.RegisterGuacdServer(server, &api.GRPCServer{})

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.WithError(err).Fatal("Failed to listen for gRPC")
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			log.WithError(err).Fatal("gRPC server failed")
		}
	}()

	return server
}

func startHTTPServer(address string) *http.Server {
	if address == "" {
		log.Info("HTTP server disabled")
		return nil
	}

	log.WithField("监听地址", address).Info("正在启动 HTTP 服务器")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Setup profiling in debug mode
	pprof.Register(router)

	// Setup routes
	controllers.SetupRoutes(router.Group("/"))

	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("HTTP server failed")
		}
	}()

	return server
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到关闭信号，正在优雅关闭...")
}
