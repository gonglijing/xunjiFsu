import { postJSON, unwrapData } from '../api';

export async function modbusSerialDebug(payload) {
  const res = await postJSON('/api/debug/modbus/serial', payload);
  return unwrapData(res, {});
}

export async function modbusTCPDebug(payload) {
  const res = await postJSON('/api/debug/modbus/tcp', payload);
  return unwrapData(res, {});
}
