import { useEffect } from 'preact/hooks';

export function useToast() {
  return (type, title, message) => {
    if (window.showToast) return window.showToast(type, title, message);
    alert(`${title || type}: ${message || ''}`);
  };
}
