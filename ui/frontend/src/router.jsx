import { createSignal } from 'solid-js';

// 路由状态
const [path, setPath] = createSignal(window.location.pathname || '/');
const [query, setQuery] = createSignal(new URLSearchParams(window.location.search));

let popstateBound = false;

// 初始化 URL 监听
if (typeof window !== 'undefined' && !popstateBound) {
  const handler = () => {
    setPath(window.location.pathname || '/');
    setQuery(new URLSearchParams(window.location.search));
  };
  window.addEventListener('popstate', handler);
  popstateBound = true;
}

// 导航函数
export function navigate(to, options = {}) {
  const target = new URL(to, window.location.origin);
  const nextPath = target.pathname || '/';
  const nextSearch = target.search || '';

  const currentPath = path();
  const currentSearch = window.location.search || '';
  if (nextPath === currentPath && nextSearch === currentSearch) return;

  const nextURL = `${nextPath}${nextSearch}`;
  
  if (options.replace) {
    window.history.replaceState({}, '', nextURL);
  } else {
    window.history.pushState({}, '', nextURL);
  }
  
  setPath(nextPath);
  setQuery(new URLSearchParams(nextSearch));
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

// 获取当前路径
export function usePath() {
  return [path, navigate, query];
}

// 获取当前路径（简化版）
export function getPath() {
  return path();
}

// 路由匹配工具
export function isActive(current, target) {
  if (target === '/') return current === '/';
  return String(current).startsWith(target);
}
