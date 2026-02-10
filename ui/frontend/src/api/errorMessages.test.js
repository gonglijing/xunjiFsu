import test from 'node:test';
import assert from 'node:assert/strict';

import { getErrorMessage, resolveAPIErrorMessage } from './errorMessages.js';

test('resolveAPIErrorMessage 优先使用 message', () => {
  const got = resolveAPIErrorMessage('E_DEVICE_NOT_FOUND', '自定义错误', 'fallback');
  assert.equal(got, '自定义错误');
});

test('resolveAPIErrorMessage 回退到 code 映射', () => {
  const got = resolveAPIErrorMessage('E_DEVICE_NOT_FOUND', '', 'fallback');
  assert.equal(got, '设备不存在');
});

test('resolveAPIErrorMessage 未命中映射时使用 fallback', () => {
  const got = resolveAPIErrorMessage('E_UNKNOWN', '', 'fallback');
  assert.equal(got, 'fallback');
});

test('getErrorMessage 处理字符串错误', () => {
  assert.equal(getErrorMessage('直接错误', 'fallback'), '直接错误');
});

test('getErrorMessage 优先 userMessage', () => {
  const got = getErrorMessage({ userMessage: '用户提示', message: '内部消息', code: 'E_DEVICE_NOT_FOUND' }, 'fallback');
  assert.equal(got, '用户提示');
});

test('getErrorMessage 其次 message', () => {
  const got = getErrorMessage({ message: '后端消息', code: 'E_DEVICE_NOT_FOUND' }, 'fallback');
  assert.equal(got, '后端消息');
});

test('getErrorMessage 再次 code 映射', () => {
  const got = getErrorMessage({ code: 'E_CLEANUP_BY_POLICY_FAILED' }, 'fallback');
  assert.equal(got, '按保留天数清理失败');
});

test('getErrorMessage 支持历史测点参数错误映射', () => {
  const got = getErrorMessage({ code: 'E_HISTORY_POINT_QUERY_INVALID' }, 'fallback');
  assert.equal(got, '历史测点参数无效');
});

test('getErrorMessage 支持清除历史数据错误映射', () => {
  const got = getErrorMessage({ code: 'E_CLEAR_HISTORY_DATA_FAILED' }, 'fallback');
  assert.equal(got, '清除历史数据失败');
});

test('getErrorMessage 空对象回退 fallback', () => {
  const got = getErrorMessage({}, 'fallback');
  assert.equal(got, 'fallback');
});
