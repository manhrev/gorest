// A flow's set of actor ids is page-specific (client-credentials has 3, authorization-code has 4),
// so this is left as a plain string rather than a fixed union.
export type ActorId = string

export interface Outcome {
  id: string
  /** short label shown in the outcome picker */
  label: string
  kind: 'happy' | 'error'
  title: string
  detail: string
  http: string
  /** id of the next FlowStep, or null if this outcome is terminal */
  next: string | null
}

export interface FlowStep {
  id: string
  from: ActorId | null
  to: ActorId | null
  outcomes: Outcome[]
}

export type FlowGraph = Record<string, FlowStep>

export interface PathEntry {
  stepId: string
  outcomeId: string
}
