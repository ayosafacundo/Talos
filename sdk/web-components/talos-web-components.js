const sharedStyle = `
:host { box-sizing: border-box; font-family: var(--talos-component-font); color: var(--talos-component-fg); }
* { box-sizing: border-box; }
`;

function ensure(name, ctor) {
  if (!customElements.get(name)) {
    customElements.define(name, ctor);
  }
}

class TalosCard extends HTMLElement {
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        ${sharedStyle}
        .card {
          border: 1px solid var(--talos-component-border);
          border-radius: var(--talos-radius-md, 12px);
          background: var(--talos-component-bg);
          padding: var(--talos-space-4, 1rem);
          box-shadow: var(--talos-component-shadow, none);
          display: grid;
          gap: var(--talos-space-3, 0.75rem);
        }
      </style>
      <section class="card" part="base">
        <header part="header"><slot name="header"></slot></header>
        <div part="content"><slot></slot></div>
        <footer part="footer"><slot name="footer"></slot></footer>
      </section>
    `;
  }
}

class TalosPanel extends HTMLElement {
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        ${sharedStyle}
        .panel {
          display: grid;
          gap: var(--talos-space-3, 0.75rem);
          border: 1px solid var(--talos-component-border);
          border-radius: var(--talos-radius-lg, 16px);
          background: var(--talos-component-bg-soft);
          padding: var(--talos-space-4, 1rem);
        }
      </style>
      <section class="panel"><slot></slot></section>
    `;
  }
}

class TalosButton extends HTMLElement {
  static get observedAttributes() {
    return ["disabled", "variant", "size", "tone"];
  }
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        ${sharedStyle}
        button {
          border: 1px solid var(--talos-component-border, transparent);
          border-radius: var(--talos-radius-sm, 8px);
          padding: 0.5rem 0.8rem;
          background: var(--talos-component-accent);
          color: var(--talos-component-button-fg, white);
          cursor: pointer;
          transition: background 120ms ease, border-color 120ms ease, opacity 120ms ease;
        }
        button[data-variant="ghost"] {
          border-color: var(--talos-component-border);
          background: transparent;
          color: var(--talos-component-fg);
        }
        button[data-size="sm"] { font-size: 0.85rem; padding: 0.35rem 0.55rem; }
        button[data-size="lg"] { font-size: 1rem; padding: 0.7rem 1rem; }
        button[data-tone="danger"] { background: var(--talos-component-danger, #d64545); }
        button[data-tone="success"] { background: var(--talos-component-success, #2f9e63); }
        button:disabled {
          cursor: not-allowed;
          opacity: 0.6;
        }
      </style>
      <button type="button" part="button"><slot></slot></button>
    `;
    this.btn = root.querySelector("button");
    this.btn.addEventListener("click", (evt) => {
      if (this.disabled) {
        evt.preventDefault();
        evt.stopPropagation();
        return;
      }
      this.dispatchEvent(new Event("click", { bubbles: true, composed: true }));
    });
  }
  get disabled() {
    return this.hasAttribute("disabled");
  }
  /** React sets boolean props as properties; getter-only `disabled` throws on WebKit. */
  set disabled(v) {
    if (v) {
      this.setAttribute("disabled", "");
    } else {
      this.removeAttribute("disabled");
    }
  }
  get variant() {
    return this.getAttribute("variant") || "solid";
  }
  set variant(v) {
    if (v == null || v === "") {
      this.removeAttribute("variant");
    } else {
      this.setAttribute("variant", String(v));
    }
  }
  get size() {
    return this.getAttribute("size") || "md";
  }
  set size(v) {
    if (v == null || v === "") {
      this.removeAttribute("size");
    } else {
      this.setAttribute("size", String(v));
    }
  }
  get tone() {
    return this.getAttribute("tone") || "accent";
  }
  set tone(v) {
    if (v == null || v === "") {
      this.removeAttribute("tone");
    } else {
      this.setAttribute("tone", String(v));
    }
  }
  attributeChangedCallback() {
    this.sync();
  }
  connectedCallback() {
    this.sync();
  }
  sync() {
    this.btn.disabled = this.disabled;
    this.btn.dataset.variant = this.getAttribute("variant") || "solid";
    this.btn.dataset.size = this.getAttribute("size") || "md";
    this.btn.dataset.tone = this.getAttribute("tone") || "accent";
  }
}

class TalosInput extends HTMLElement {
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        ${sharedStyle}
        input {
          width: 100%;
          border: 1px solid var(--talos-component-border);
          border-radius: var(--talos-radius-sm, 8px);
          background: var(--talos-component-bg);
          color: var(--talos-component-fg);
          padding: 0.5rem 0.65rem;
        }
      </style>
      <input />
    `;
    this.input = root.querySelector("input");
    this.input.addEventListener("input", () => {
      this.value = this.input.value;
      this.dispatchEvent(new Event("input", { bubbles: true, composed: true }));
    });
  }
  connectedCallback() {
    const placeholder = this.getAttribute("placeholder");
    if (placeholder) {
      this.input.placeholder = placeholder;
    }
    this.input.value = this.getAttribute("value") || "";
  }
  get value() {
    return this.getAttribute("value") || "";
  }
  set value(v) {
    this.setAttribute("value", String(v ?? ""));
  }
}

class TalosAlert extends HTMLElement {
  static get observedAttributes() {
    return ["tone"];
  }
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        ${sharedStyle}
        .alert {
          border: 1px solid var(--talos-component-border);
          border-left: 4px solid var(--talos-component-accent);
          border-radius: var(--talos-radius-sm, 8px);
          background: var(--talos-component-bg-soft);
          padding: 0.65rem 0.8rem;
        }
        .title {
          margin: 0 0 0.25rem 0;
          font-weight: 600;
        }
        .body {
          color: var(--talos-component-muted);
        }
        .alert[data-tone="danger"] { border-left-color: var(--talos-component-danger, #d64545); }
        .alert[data-tone="success"] { border-left-color: var(--talos-component-success, #2f9e63); }
      </style>
      <section class="alert" part="base">
        <p class="title" part="title"><slot name="title"></slot></p>
        <div class="body" part="body"><slot></slot></div>
      </section>
    `;
    this.base = root.querySelector(".alert");
  }
  connectedCallback() {
    this.sync();
  }
  attributeChangedCallback() {
    this.sync();
  }
  sync() {
    this.base.dataset.tone = this.getAttribute("tone") || "accent";
  }
  get tone() {
    return this.getAttribute("tone") || "accent";
  }
  set tone(v) {
    if (v == null || v === "") {
      this.removeAttribute("tone");
    } else {
      this.setAttribute("tone", String(v));
    }
  }
}

class TalosListRow extends HTMLElement {
  constructor() {
    super();
    const root = this.attachShadow({ mode: "open" });
    root.innerHTML = `
      <style>
        ${sharedStyle}
        .row {
          display: grid;
          grid-template-columns: auto 1fr auto;
          align-items: center;
          gap: var(--talos-space-3, 0.75rem);
          border-bottom: 1px solid var(--talos-component-border);
          padding: 0.6rem 0;
        }
        .row:hover { background: var(--talos-component-row-hover, transparent); }
        .meta { color: var(--talos-component-muted); font-size: 0.9em; }
      </style>
      <div class="row" part="row">
        <slot name="leading" part="leading"></slot>
        <div part="content">
          <slot part="text"></slot>
          <div class="meta" part="meta"><slot name="meta"></slot></div>
        </div>
        <slot name="trailing" part="trailing"></slot>
      </div>
    `;
  }
}

export function registerTalosWebComponents() {
  ensure("talos-card", TalosCard);
  ensure("talos-panel", TalosPanel);
  ensure("talos-button", TalosButton);
  ensure("talos-input", TalosInput);
  ensure("talos-alert", TalosAlert);
  ensure("talos-list-row", TalosListRow);
}
