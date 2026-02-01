package app

import (
    "time"

    "github.com/gonglijing/xunjiFsu/internal/database"
    "github.com/gonglijing/xunjiFsu/internal/logger"
    "github.com/gonglijing/xunjiFsu/internal/northbound"
)

// startNorthboundSchedulers 根据数据库配置设置北向上传周期与启停
func startNorthboundSchedulers(nm *northbound.NorthboundManager) {
    configs, err := database.GetAllNorthboundConfigs()
    if err != nil {
        logger.Warn("Failed to load northbound configs", "error", err)
        return
    }

    for _, cfg := range configs {
        nm.SetInterval(cfg.Name, time.Duration(cfg.UploadInterval)*time.Millisecond)
        nm.SetEnabled(cfg.Name, cfg.Enabled == 1)
    }
}
