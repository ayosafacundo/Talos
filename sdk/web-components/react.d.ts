import type { TalosAppRuntime } from "./index"
import type { DetailedHTMLProps, HTMLAttributes, ReactNode } from "react"

export function useTalosApp(appID: string): TalosAppRuntime & { themeReady: boolean }

export type TalosComponentProps = DetailedHTMLProps<HTMLAttributes<HTMLElement>, HTMLElement> & {
  variant?: string
  size?: "sm" | "md" | "lg"
  tone?: "accent" | "success" | "danger"
  children?: ReactNode
}

export function TalosPanel(props: TalosComponentProps): ReactNode
export function TalosCard(props: TalosComponentProps): ReactNode
export function TalosButton(props: TalosComponentProps): ReactNode
export function TalosAlert(props: TalosComponentProps): ReactNode
export function TalosListRow(props: TalosComponentProps): ReactNode
