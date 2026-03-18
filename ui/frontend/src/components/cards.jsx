function Card(props) {
  return (
    <div class="card" style={props.style || ''}>
      {(props.title || props.extra) && (
        <div class="card-header">
          {props.title && <h3 class="card-title">{props.title}</h3>}
          {props.extra && <div class="card-extra">{props.extra}</div>}
        </div>
      )}
      <div class="card-body">
        {props.children}
      </div>
    </div>
  );
}

export default Card;
