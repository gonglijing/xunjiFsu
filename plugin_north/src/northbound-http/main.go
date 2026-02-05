package main

import (
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	plugin "github.com/hashicorp/go-plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: northbound.NorthboundHandshake,
		Plugins: map[string]plugin.Plugin{
			northbound.NorthboundPluginName: &northbound.NorthboundPlugin{Impl: northbound.NewHTTPAdapter()},
		},
	})
}
