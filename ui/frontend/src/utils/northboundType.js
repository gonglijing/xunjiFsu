export const NORTHBOUND_TYPE = Object.freeze({
  MQTT: 'mqtt',
  PANDAX: 'pandax',
  ITHINGS: 'ithings',
  SAGOO: 'sagoo',
});

export function normalizeNorthboundType(type) {
  return `${type ?? ''}`.trim().toLowerCase();
}

export function isSchemaDrivenType(type) {
  const normalized = normalizeNorthboundType(type);
  return normalized === NORTHBOUND_TYPE.SAGOO
    || normalized === NORTHBOUND_TYPE.PANDAX
    || normalized === NORTHBOUND_TYPE.ITHINGS;
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
