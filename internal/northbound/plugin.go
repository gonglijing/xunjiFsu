package northbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	northboundpb "github.com/gonglijing/xunjiFsu/internal/northbound/pb"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	grpcConn   *grpc.ClientConn
	xunjiGRPC  northboundpb.XunjiIngressClient
	grpcTarget string
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
	if adapter.SelfManaged() {
		if err := adapter.initXunJiGRPC(config); err != nil {
			_ = adapter.Close()
			return nil, err
		}
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
	if a.xunjiGRPC != nil {
		if data == nil {
			return nil
		}
		timestamp := data.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}
		fields := make(map[string]string, len(data.Fields))
		for key, value := range data.Fields {
			fields[key] = value
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		resp, err := a.xunjiGRPC.PushRealtime(ctx, &northboundpb.PushRealtimeRequest{
			DeviceId:        data.DeviceID,
			DeviceName:      data.DeviceName,
			ProductKey:      data.ProductKey,
			DeviceKey:       data.DeviceKey,
			TimestampUnixMs: timestamp.UnixMilli(),
			Fields:          fields,
		})
		if err != nil {
			return err
		}
		if resp != nil && !resp.GetSuccess() {
			return fmt.Errorf("xunji grpc push realtime rejected: %s", resp.GetMessage())
		}
		return nil
	}

	if a.impl == nil {
		return errors.New("plugin not initialized")
	}
	return a.impl.Send(data)
}

func (a *PluginAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if a.xunjiGRPC != nil {
		if alarm == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		resp, err := a.xunjiGRPC.PushAlarm(ctx, &northboundpb.PushAlarmRequest{
			DeviceId:        alarm.DeviceID,
			DeviceName:      alarm.DeviceName,
			ProductKey:      alarm.ProductKey,
			DeviceKey:       alarm.DeviceKey,
			FieldName:       alarm.FieldName,
			ActualValue:     alarm.ActualValue,
			Threshold:       alarm.Threshold,
			Operator:        alarm.Operator,
			Severity:        alarm.Severity,
			Message:         alarm.Message,
			TriggeredUnixMs: time.Now().UnixMilli(),
		})
		if err != nil {
			return err
		}
		if resp != nil && !resp.GetSuccess() {
			return fmt.Errorf("xunji grpc push alarm rejected: %s", resp.GetMessage())
		}
		return nil
	}

	if a.impl == nil {
		return errors.New("plugin not initialized")
	}
	return a.impl.SendAlarm(alarm)
}

func (a *PluginAdapter) Close() error {
	if a.grpcConn != nil {
		_ = a.grpcConn.Close()
	}
	a.grpcConn = nil
	a.xunjiGRPC = nil
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

// SelfManaged indicates whether this plugin handles send timing internally.
func (a *PluginAdapter) SelfManaged() bool {
	return strings.EqualFold(strings.TrimSpace(a.pluginType), "xunji")
}

func (a *PluginAdapter) initXunJiGRPC(config string) error {
	address, err := resolveXunJiGRPCAddress(config)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("dial xunji grpc %s failed: %w", address, err)
	}
	a.grpcConn = conn
	a.xunjiGRPC = northboundpb.NewXunjiIngressClient(conn)
	a.grpcTarget = address
	return nil
}

func resolveXunJiGRPCAddress(config string) (string, error) {
	parsed := &models.XunJiConfig{}
	if err := json.Unmarshal([]byte(config), parsed); err != nil {
		return "", fmt.Errorf("parse xunji config failed: %w", err)
	}
	if v := strings.TrimSpace(parsed.GRPCAddress); v != "" {
		return v, nil
	}
	pk := strings.TrimSpace(parsed.ProductKey)
	dk := strings.TrimSpace(parsed.DeviceKey)
	if pk == "" || dk == "" {
		return "", fmt.Errorf("xunji grpc address resolve failed: productKey/deviceKey required")
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(pk + "|" + dk))
	port := 30000 + int(hasher.Sum32()%10000)
	return "127.0.0.1:" + strconv.Itoa(port), nil
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
