export function SectionTabs({ tabs, active, onChange }) {
  return (
    <div class="tabs mb-6">
      {tabs.map((t) => (
        <button
          key={t.id}
          class={`tab-btn ${active === t.id ? 'active' : ''}`}
          onClick={() => onChange(t.id)}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}
