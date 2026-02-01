export const Card = ({ title, extra, children }) => (
  <div class="card" style="margin-bottom:16px;">
    <div class="card-header">
      <h3 class="card-title">{title}</h3>
      {extra && <div>{extra}</div>}
    </div>
    {children}
  </div>
);
