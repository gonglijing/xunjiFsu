export function formatDateTime(value) {
  if (!value) return '';
  let date;
  if (value instanceof Date) {
    date = value;
  } else if (typeof value === 'number') {
    date = new Date(value);
  } else if (typeof value === 'string') {
    const trimmed = value.trim();
    let normalized = trimmed;
    if (trimmed.includes(' ') && !trimmed.includes('T')) {
      const idx = trimmed.indexOf(' ');
      normalized = trimmed.slice(0, idx) + 'T' + trimmed.slice(idx + 1);
    }
    date = new Date(normalized);
  }

  if (!date || Number.isNaN(date.getTime())) {
    return String(value);
  }

  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}/${pad(date.getMonth() + 1)}/${pad(date.getDate())} ` +
    `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}
