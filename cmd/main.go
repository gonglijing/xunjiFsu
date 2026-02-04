package main

import (
	"flag"
	"log"

	"github.com/gonglijing/xunjiFsu/internal/app"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	setupLogger(cfg)

	if err := app.Run(cfg); err != nil {
		logger.Fatal("Application error", err)
	}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "database path")
	flag.StringVar(&cfg.ListenAddr, "addr", cfg.ListenAddr, "listen address")
	flag.Parse()

	return cfg, nil
}

func setupLogger(cfg *config.Config) {
	level := logger.ParseLevel(cfg.LogLevel)
	logger.SetLevel(level)
	logger.SetJSONOutput(cfg.LogJSON)
	if _, err := logger.InitFileOutput("logs/xunji.log", 2*1024*1024); err != nil {
		log.Printf("Failed to init file logger: %v", err)
	}
}
