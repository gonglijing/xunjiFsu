export const NORTHBOUND_TYPE = Object.freeze({
  MQTT: 'mqtt',
  PANDAX: 'pandax',
  ITHINGS: 'ithings',
  SAGOO: 'sagoo',
  LEGACY_XUNJI: 'xunji',
});

export function normalizeNorthboundType(type) {
  const raw = `${type ?? ''}`.trim().toLowerCase();
  if (raw === NORTHBOUND_TYPE.LEGACY_XUNJI) return NORTHBOUND_TYPE.SAGOO;
  return raw;
}

export function isSagooType(type) {
  return normalizeNorthboundType(type) === NORTHBOUND_TYPE.SAGOO;
}

export function getNorthboundTypeLabel(type) {
  switch (normalizeNorthboundType(type)) {
    case NORTHBOUND_TYPE.MQTT:
      return 'MQTT';
    case NORTHBOUND_TYPE.PANDAX:
      return 'PandaX';
    case NORTHBOUND_TYPE.ITHINGS:
      return 'iThings';
    case NORTHBOUND_TYPE.SAGOO:
      return 'Sagoo';
    default:
      return `${type ?? ''}`.trim().toUpperCase();
  }
}

