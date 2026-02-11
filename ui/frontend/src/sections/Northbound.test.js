import test from 'node:test';
import assert from 'node:assert/strict';

import {
  fillPayloadFromConfig,
  getNorthboundServerAddress,
  parseConfigFromItem,
  toUploadIntervalMs,
  toUploadIntervalSeconds,
} from './northboundFormHelpers.js';

test('toUploadIntervalSeconds 把毫秒换算为秒', () => {
  assert.equal(toUploadIntervalSeconds(5000, 5), 5);
  assert.equal(toUploadIntervalSeconds(12000, 5), 12);
});

test('toUploadIntervalMs 把秒换算为毫秒', () => {
  assert.equal(toUploadIntervalMs(5, 5000), 5000);
  assert.equal(toUploadIntervalMs('12', 5000), 12000);
});

test('parseConfigFromItem 回填 MQTT 地址与上传周期', () => {
  const cfg = parseConfigFromItem({
    type: 'mqtt',
    server_url: 'tcp://1.2.3.4:1883',
    config: '{}',
  }, 'mqtt', 7000);

  assert.equal(cfg.broker, 'tcp://1.2.3.4:1883');
  assert.equal(cfg.uploadIntervalMs, 7000);
});

test('fillPayloadFromConfig 从 schema 配置提取 server_url', () => {
  const payload = {};
  fillPayloadFromConfig(payload, { serverUrl: 'tcp://127.0.0.1:1883' });
  assert.equal(payload.server_url, 'tcp://127.0.0.1:1883');
});

test('getNorthboundServerAddress 优先显示 server_url', () => {
  const got = getNorthboundServerAddress({ server_url: 'tcp://10.0.0.2:1883' });
  assert.equal(got, 'tcp://10.0.0.2:1883');
});

test('getNorthboundServerAddress 回退解析 config.broker', () => {
  const got = getNorthboundServerAddress({ config: '{"broker":"tcp://127.0.0.1:1883"}' });
  assert.equal(got, 'tcp://127.0.0.1:1883');
});
