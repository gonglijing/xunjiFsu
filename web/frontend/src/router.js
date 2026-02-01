import { useEffect, useState } from 'preact/hooks';

export function usePath() {
  const [path, setPath] = useState(() => window.location.pathname || '/');

  useEffect(() => {
    const handler = () => setPath(window.location.pathname || '/');
    window.addEventListener('popstate', handler);
    return () => window.removeEventListener('popstate', handler);
  }, []);

  const navigate = (to) => {
    if (to === path) return;
    window.history.pushState({}, '', to);
    setPath(to);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return [path, navigate];
}

export function isActive(current, target) {
  if (target === '/') return current === '/';
  return current.startsWith(target);
}
