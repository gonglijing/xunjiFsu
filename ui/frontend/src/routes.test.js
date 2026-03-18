import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveRoute, isLoginRoute, mainLinks, settingsLinks, debugLinks } from './route-config.js';

test('resolveRoute matches exact login route', () => {
  const route = resolveRoute('/login');
  assert.equal(route.key, 'login');
  assert.equal(route.layout, 'login');
});

test('resolveRoute prefers specific debug routes before generic debug route', () => {
  assert.equal(resolveRoute('/debug/modbus-serial').key, 'debug-modbus-serial');
  assert.equal(resolveRoute('/debug/modbus-tcp').key, 'debug-modbus-tcp');
  assert.equal(resolveRoute('/debug/tools').key, 'debug-tools');
});

test('resolveRoute falls back to topology for unknown paths', () => {
  const route = resolveRoute('/unknown/path');
  assert.equal(route.key, 'home');
  assert.equal(route.layout, 'shell');
});

test('isLoginRoute only returns true for login layout', () => {
  assert.equal(isLoginRoute('/login'), true);
  assert.equal(isLoginRoute('/devices'), false);
  assert.equal(isLoginRoute('/missing'), false);
});

test('navigation groups are derived from route config', () => {
  assert.deepEqual(
    mainLinks.map((route) => route.path),
    ['/', '/alarms', '/realtime'],
  );
  assert.deepEqual(
    settingsLinks.map((route) => route.path),
    ['/gateway', '/resources', '/devices', '/drivers', '/northbound', '/thresholds'],
  );
  assert.deepEqual(
    debugLinks.map((route) => route.path),
    ['/debug/modbus-serial', '/debug/modbus-tcp'],
  );
});
