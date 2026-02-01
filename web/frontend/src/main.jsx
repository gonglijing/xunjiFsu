import { render } from 'preact';
import { App } from './App';
import { Devices } from './islands/Devices';

const islands = { devices: Devices };

document.addEventListener('DOMContentLoaded', () => {
  const root = document.getElementById('app-root');
  if (root) {
    render(<App />, root);
    return;
  }
  document.querySelectorAll('[data-preact-island]').forEach((el) => {
    const name = el.getAttribute('data-preact-island');
    const Component = islands[name];
    if (!Component) return;
    const props = el.dataset.props ? JSON.parse(el.dataset.props) : {};
    render(<Component {...props} />, el);
  });
});
