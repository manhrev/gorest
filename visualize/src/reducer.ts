import type { ActorId, FlowGraph, PathEntry } from './types'
import { START_STEP } from './flowGraph'

export interface FlowState {
  path: PathEntry[]
  cursor: number
  openActor: ActorId | null
  playing: boolean
}

export type Action =
  | { type: 'CHOOSE_OUTCOME'; outcomeId: string }
  | { type: 'NEXT' }
  | { type: 'PREV' }
  | { type: 'RESET' }
  | { type: 'PLAY_TICK' }
  | { type: 'TOGGLE_PLAY' }
  | { type: 'STOP_PLAY' }
  | { type: 'OPEN_ACTOR'; actor: ActorId }
  | { type: 'CLOSE_ACTOR' }

const defaultOutcomeId = (graph: FlowGraph, stepId: string) =>
  graph[stepId].outcomes.find((o) => o.kind === 'happy')?.id ?? graph[stepId].outcomes[0].id

/**
 * Total step count for the "Step X / N" display. `path.length` alone is wrong here —
 * path only grows as the user clicks Next, so cursor is always its last index while
 * moving forward and it reads as "N/N" no matter where you are. Instead, project the
 * path forward from wherever it currently ends, using the default outcome at each
 * not-yet-visited step, until a terminal step is hit.
 */
export function totalSteps(graph: FlowGraph, path: PathEntry[]): number {
  let count = path.length
  let last = path[path.length - 1]
  let outcome = graph[last.stepId].outcomes.find((o) => o.id === last.outcomeId)!
  while (outcome.next) {
    count++
    const nextStep = graph[outcome.next]
    outcome = nextStep.outcomes.find((o) => o.kind === 'happy') ?? nextStep.outcomes[0]
  }
  return count
}

export function initState(): FlowState {
  return { path: [{ stepId: START_STEP, outcomeId: 'happy' }], cursor: 0, openActor: null, playing: false }
}

export function makeReducer(graph: FlowGraph) {
  return function reducer(state: FlowState, action: Action): FlowState {
    switch (action.type) {
      case 'CHOOSE_OUTCOME': {
        const { stepId } = state.path[state.cursor]
        const path = state.path.slice(0, state.cursor + 1)
        path[state.cursor] = { stepId, outcomeId: action.outcomeId }
        return { ...state, path, playing: false }
      }
      case 'NEXT': {
        if (state.cursor + 1 < state.path.length) {
          return { ...state, cursor: state.cursor + 1 }
        }
        const { stepId, outcomeId } = state.path[state.cursor]
        const outcome = graph[stepId].outcomes.find((o) => o.id === outcomeId)!
        if (!outcome.next) return { ...state, playing: false } // terminal
        const nextStepId = outcome.next
        const path = [...state.path, { stepId: nextStepId, outcomeId: defaultOutcomeId(graph, nextStepId) }]
        return { ...state, path, cursor: state.cursor + 1 }
      }
      case 'PLAY_TICK': {
        // same as NEXT, but stops playback instead of no-op at a terminal step
        const { stepId, outcomeId } = state.path[state.cursor]
        const outcome = graph[stepId].outcomes.find((o) => o.id === outcomeId)!
        if (!outcome.next) return { ...state, playing: false }
        const nextStepId = outcome.next
        const path = [...state.path, { stepId: nextStepId, outcomeId: defaultOutcomeId(graph, nextStepId) }]
        return { ...state, path, cursor: state.cursor + 1 }
      }
      case 'PREV':
        return { ...state, cursor: Math.max(0, state.cursor - 1), playing: false }
      case 'RESET':
        return { ...initState() }
      case 'TOGGLE_PLAY':
        return { ...state, playing: !state.playing }
      case 'STOP_PLAY':
        return { ...state, playing: false }
      case 'OPEN_ACTOR':
        return { ...state, openActor: state.openActor === action.actor ? null : action.actor }
      case 'CLOSE_ACTOR':
        return { ...state, openActor: null }
      default:
        return state
    }
  }
}
