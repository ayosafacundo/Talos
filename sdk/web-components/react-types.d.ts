import type { CSSProperties, DetailedHTMLProps, HTMLAttributes } from "react"

type TalosElementProps = DetailedHTMLProps<HTMLAttributes<HTMLElement>, HTMLElement> & {
  style?: CSSProperties
  variant?: string
}

declare global {
  namespace JSX {
    interface IntrinsicElements {
      "talos-panel": TalosElementProps;
      "talos-card": TalosElementProps;
      "talos-button": TalosElementProps;
      "talos-alert": TalosElementProps;
      "talos-list-row": TalosElementProps;
    }
  }
}

declare module "react" {
  namespace JSX {
    interface IntrinsicElements {
      "talos-panel": TalosElementProps;
      "talos-card": TalosElementProps;
      "talos-button": TalosElementProps;
      "talos-alert": TalosElementProps;
      "talos-list-row": TalosElementProps;
    }
  }
}

export {}
