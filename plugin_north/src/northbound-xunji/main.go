package main

import (
	plugin "github.com/hashicorp/go-plugin"

	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/plugin_north/adapter"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: northbound.NorthboundHandshake,
		Plugins: map[string]plugin.Plugin{
			northbound.NorthboundPluginName: &northbound.NorthboundPlugin{Impl: adapter.NewXunJiAdapter()},
		},
	})
}
