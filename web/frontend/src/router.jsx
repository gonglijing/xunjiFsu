import { createSignal, createEffect, onCleanup, Show, For } from 'solid-js';

// 路由状态
const [path, setPath] = createSignal(window.location.pathname || '/');
const [query, setQuery] = createSignal(new URLSearchParams(window.location.search));

// 初始化 URL 监听
if (typeof window !== 'undefined') {
  const handler = () => {
    setPath(window.location.pathname || '/');
    setQuery(new URLSearchParams(window.location.search));
  };
  window.addEventListener('popstate', handler);
  onCleanup(() => window.removeEventListener('popstate', handler));
}

// 导航函数
export function navigate(to, options = {}) {
  const current = path();
  if (to === current) return;
  
  if (options.replace) {
    window.history.replaceState({}, '', to);
  } else {
    window.history.pushState({}, '', to);
  }
  
  setPath(to);
  setQuery(new URLSearchParams(window.location.search));
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
