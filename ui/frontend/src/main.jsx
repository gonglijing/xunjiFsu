import { render } from 'solid-js/web';
import App from './App';
import { logFrontendRuntimeConfig } from './utils/runtimeConfig';

// SolidJS 入口渲染
const root = document.getElementById('app-root');

logFrontendRuntimeConfig();

if (root) {
  render(() => <App />, root);
}
