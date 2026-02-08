import { render } from 'solid-js/web';
import App from './App';
import { logFrontendRuntimeConfig } from './utils/runtimeConfig';

// SolidJS 入口渲染
const root = document.getElementById('app-root');

logFrontendRuntimeConfig();

if (root) {
  render(() => <App />, root);
}

// Islands 模式支持
const islands = {
  // 可以在此添加需要独立渲染的 islands
};

// DOM 加载后初始化 islands
if (typeof document !== 'undefined') {
  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-solid-island]').forEach((el) => {
      const name = el.getAttribute('data-solid-island');
      const Component = islands[name];
      if (!Component) return;
      
      const props = el.dataset.props ? JSON.parse(el.dataset.props) : {};
      render(() => <Component {...props} />, el);
    });
  });
}
