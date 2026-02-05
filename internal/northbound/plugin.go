package northbound

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	plugin "github.com/hashicorp/go-plugin"
)

const NorthboundPluginName = "northbound"

var NorthboundHandshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GOGW_NORTHBOUND_PLUGIN",
	MagicCookieValue: "1",
}

type NorthboundPlugin struct {
	Impl Northbound
}

func (p *NorthboundPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &NorthboundRPCServer{Impl: p.Impl}, nil
}

func (p *NorthboundPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &NorthboundRPCClient{client: c}, nil
}

// PluginAdapter wraps a northbound plugin client.
type PluginAdapter struct {
	name       string
	pluginType string
	client     *plugin.Client
	impl       Northbound
	pluginPath string
}

func NewPluginAdapter(pluginDir, pluginType, name, config string) (*PluginAdapter, error) {
	path, err := resolvePluginPath(pluginDir, pluginType, config)
	if err != nil {
		return nil, err
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: NorthboundHandshake,
		Plugins: map[string]plugin.Plugin{
			NorthboundPluginName: &NorthboundPlugin{},
		},
		Cmd:              exec.Command(path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}

	raw, err := rpcClient.Dispense(NorthboundPluginName)
	if err != nil {
		client.Kill()
		return nil, err
	}

	impl, ok := raw.(Northbound)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("unexpected plugin type %T", raw)
	}

	adapter := &PluginAdapter{
		name:       name,
		pluginType: pluginType,
		client:     client,
		impl:       impl,
		pluginPath: path,
	}

	if err := impl.Initialize(config); err != nil {
		_ = adapter.Close()
		return nil, err
	}

	return adapter, nil
}

func (a *PluginAdapter) Initialize(config string) error {
	if a.impl == nil {
		return errors.New("plugin not initialized")
	}
	return a.impl.Initialize(config)
}

func (a *PluginAdapter) Send(data *models.CollectData) error {
	if a.impl == nil {
		return errors.New("plugin not initialized")
	}
	return a.impl.Send(data)
}

func (a *PluginAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if a.impl == nil {
		return errors.New("plugin not initialized")
	}
	return a.impl.SendAlarm(alarm)
}

func (a *PluginAdapter) Close() error {
	if a.impl != nil {
		_ = a.impl.Close()
	}
	if a.client != nil {
		a.client.Kill()
	}
	a.impl = nil
	return nil
}

func (a *PluginAdapter) Name() string {
	if a.impl != nil {
		return a.impl.Name()
	}
	if a.name != "" {
		return a.name
	}
	return a.pluginType
}

// resolvePluginPath resolves the northbound plugin binary.
func resolvePluginPath(pluginDir, pluginType, config string) (string, error) {
	if pluginType == "" {
		return "", fmt.Errorf("northbound type is empty")
	}
	if path := extractPluginPath(config); path != "" {
		return path, nil
	}
	if pluginDir == "" {
		pluginDir = "plugin_north"
	}

	candidates := []string{
		filepath.Join(pluginDir, fmt.Sprintf("northbound-%s", pluginType)),
		filepath.Join(pluginDir, pluginType),
	}

	for _, candidate := range candidates {
		path := candidate
		if runtime.GOOS == "windows" && filepath.Ext(path) == "" {
			path += ".exe"
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	fallback := candidates[0]
	if runtime.GOOS == "windows" && filepath.Ext(fallback) == "" {
		fallback += ".exe"
	}
	return "", fmt.Errorf("northbound plugin not found for type %s (expected %s)", pluginType, fallback)
}

func extractPluginPath(config string) string {
	if config == "" {
		return ""
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		return ""
	}
	keys := []string{"plugin_path", "pluginPath", "plugin", "binary", "bin"}
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// NewAdapterFromConfig creates a plugin-backed adapter from config.
func NewAdapterFromConfig(pluginDir string, config *models.NorthboundConfig) (Northbound, error) {
	if config == nil {
		return nil, fmt.Errorf("northbound config is nil")
	}
	adapter, err := NewPluginAdapter(pluginDir, config.Type, config.Name, config.Config)
	if err != nil {
		logger.Warn("Northbound plugin load failed", "name", config.Name, "type", config.Type, "error", err)
		return nil, err
	}
	return adapter, nil
}

// RPC structs

type initArgs struct {
	Config string
}

type sendArgs struct {
	Data *models.CollectData
}

type alarmArgs struct {
	Alarm *models.AlarmPayload
}

type nameResp struct {
	Name string
}

// NorthboundRPCServer implements the RPC server.
type NorthboundRPCServer struct {
	Impl Northbound
}

func (s *NorthboundRPCServer) Initialize(args *initArgs, resp *struct{}) error {
	return s.Impl.Initialize(args.Config)
}

func (s *NorthboundRPCServer) Send(args *sendArgs, resp *struct{}) error {
	return s.Impl.Send(args.Data)
}

func (s *NorthboundRPCServer) SendAlarm(args *alarmArgs, resp *struct{}) error {
	return s.Impl.SendAlarm(args.Alarm)
}

func (s *NorthboundRPCServer) Close(_ *struct{}, _ *struct{}) error {
	return s.Impl.Close()
}

func (s *NorthboundRPCServer) Name(_ *struct{}, resp *nameResp) error {
	resp.Name = s.Impl.Name()
	return nil
}

// NorthboundRPCClient implements the RPC client.
type NorthboundRPCClient struct {
	client *rpc.Client
}

func (c *NorthboundRPCClient) Initialize(config string) error {
	return c.client.Call("Plugin.Initialize", &initArgs{Config: config}, &struct{}{})
}

func (c *NorthboundRPCClient) Send(data *models.CollectData) error {
	return c.client.Call("Plugin.Send", &sendArgs{Data: data}, &struct{}{})
}

func (c *NorthboundRPCClient) SendAlarm(alarm *models.AlarmPayload) error {
	return c.client.Call("Plugin.SendAlarm", &alarmArgs{Alarm: alarm}, &struct{}{})
}

func (c *NorthboundRPCClient) Close() error {
	return c.client.Call("Plugin.Close", &struct{}{}, &struct{}{})
}

func (c *NorthboundRPCClient) Name() string {
	resp := &nameResp{}
	if err := c.client.Call("Plugin.Name", &struct{}{}, resp); err != nil {
		return ""
	}
	return resp.Name
}
