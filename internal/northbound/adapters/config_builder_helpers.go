package adapters

import (
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
)

type configDefaults struct {
	copyBrokerToServerURL bool
	values                map[string]interface{}
}

type modelBuildOptions struct {
	includeTopic           bool
	includeAlarmTopic      bool
	includeProductIdentity bool
	includeUploadInterval  bool
}

var northboundConfigDefaults = map[string]configDefaults{
	nbtype.TypeMQTT: {
		values: map[string]interface{}{
			"broker":         "",
			"topic":          "",
			"qos":            0,
			"keepAlive":      60,
			"connectTimeout": 30,
		},
	},
	nbtype.TypeXunji: {
		copyBrokerToServerURL: true,
		values: map[string]interface{}{
			"topic":            "v1/gateway/{gatewayname}",
			"qos":              0,
			"keepAlive":        60,
			"connectTimeout":   10,
			"uploadIntervalMs": 5000,
		},
	},
	nbtype.TypeSagoo: {
		values: map[string]interface{}{
			"serverUrl":        "",
			"productKey":       "",
			"deviceKey":        "",
			"qos":              0,
			"keepAlive":        60,
			"connectTimeout":   30,
			"uploadIntervalMs": 5000,
		},
	},
	nbtype.TypePandaX: {
		copyBrokerToServerURL: true,
		values: map[string]interface{}{
			"username":         "",
			"qos":              0,
			"keepAlive":        60,
			"connectTimeout":   10,
			"uploadIntervalMs": 5000,
		},
	},
	nbtype.TypeIThings: {
		copyBrokerToServerURL: true,
		values: map[string]interface{}{
			"username":         "",
			"productKey":       "",
			"deviceKey":        "",
			"gatewayMode":      true,
			"qos":              0,
			"keepAlive":        60,
			"connectTimeout":   10,
			"uploadIntervalMs": 5000,
		},
	},
}

func (b *NorthboundConfigBuilder) applyDefaults(defaults configDefaults) {
	if defaults.copyBrokerToServerURL {
		b.ensureServerURLFromBroker()
	}
	for key, value := range defaults.values {
		b.ensureConfigValue(key, value)
	}
}

func (b *NorthboundConfigBuilder) ensureServerURLFromBroker() {
	if _, ok := b.config["serverUrl"]; ok {
		return
	}
	if broker, exists := b.config["broker"]; exists {
		b.config["serverUrl"] = broker
		return
	}
	b.config["serverUrl"] = ""
}

func (b *NorthboundConfigBuilder) ensureConfigValue(key string, value interface{}) {
	if _, ok := b.config[key]; ok {
		return
	}
	b.config[key] = value
}

func applySharedModelFields(builder *NorthboundConfigBuilder, cfg *models.NorthboundConfig, options modelBuildOptions) {
	builder.SetBrokerURL(buildBrokerURL(cfg.ServerURL, cfg.Port))
	builder.SetClientID(cfg.ClientID)
	builder.SetUsername(cfg.Username)
	builder.SetPassword(cfg.Password)
	if options.includeTopic {
		builder.SetTopic(cfg.Topic)
	}
	if options.includeAlarmTopic {
		builder.SetAlarmTopic(cfg.AlarmTopic)
	}
	if options.includeProductIdentity {
		builder.SetProductKey(cfg.ProductKey)
		builder.SetDeviceKey(cfg.DeviceKey)
	}
	builder.SetQOS(cfg.QOS)
	builder.SetRetain(cfg.Retain)
	builder.SetKeepAlive(cfg.KeepAlive)
	builder.SetTimeout(cfg.Timeout)
	if options.includeUploadInterval {
		builder.SetUploadIntervalMs(cfg.UploadInterval)
	}
	builder.SetExtConfig(cfg.ExtConfig)
}

func configServerURL(config map[string]interface{}) string {
	return pickConfigString(config, "serverUrl", "broker")
}
