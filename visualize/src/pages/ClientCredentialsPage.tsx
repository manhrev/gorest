import { useCallback, useEffect, useMemo, useReducer } from 'react'
import FlowCanvas, { type ActorMeta } from '../components/FlowCanvas'
import StepPanel from '../components/StepPanel'
import OutcomePicker from '../components/OutcomePicker'
import ActorInfoPanel, { type ActorInfoContent } from '../components/ActorInfoPanel'
import PlaybackControls from '../components/PlaybackControls'
import { graph, actorInfo, clientHolding } from '../flowGraph'
import { makeReducer, initState } from '../reducer'
import type { ActorId } from '../types'

const reducer = makeReducer(graph)
const PLAY_INTERVAL_MS = 1800

const ACTOR_META: Record<string, ActorMeta> = {
  client: { icon: '🖧', label: 'OAuth Client', position: { x: 260, y: 0 } },
  auth: { icon: '🔐', label: 'Auth Server', position: { x: 0, y: 220 } },
  resource: { icon: '📦', label: 'Resource Server', position: { x: 520, y: 220 } },
}

const EDGE_PAIRS: [ActorId, ActorId][] = [
  ['client', 'auth'],
  ['client', 'resource'],
]

export default function ClientCredentialsPage() {
  const [state, dispatch] = useReducer(reducer, undefined, initState)

  const entry = state.path[state.cursor]
  const step = graph[entry.stepId]
  const outcome = step.outcomes.find((o) => o.id === entry.outcomeId)!
  const atStart = state.cursor === 0
  const atEnd = outcome.next === null

  const tokenIssued = state.path
    .slice(0, state.cursor + 1)
    .some((p) => p.stepId === 'issue-token' && p.outcomeId === 'success')
  const resourceFetched = state.path.slice(0, state.cursor + 1).some((p) => p.stepId === 'resource-returned')

  useEffect(() => {
    if (!state.playing) return
    if (atEnd) return
    const timer = setInterval(() => dispatch({ type: 'PLAY_TICK' }), PLAY_INTERVAL_MS)
    return () => clearInterval(timer)
  }, [state.playing, atEnd])

  const onActorClick = useCallback((actor: ActorId) => dispatch({ type: 'OPEN_ACTOR', actor }), [])

  const pulseKey = useMemo(() => `${entry.stepId}:${entry.outcomeId}`, [entry.stepId, entry.outcomeId])

  const getInfo = useCallback(
    (actor: ActorId): ActorInfoContent => {
      if (actor === 'client') {
        return { title: 'Client — holding', sections: [{ items: clientHolding({ tokenIssued, resourceFetched }) }] }
      }
      const info = actorInfo[actor as 'auth' | 'resource']
      return {
        title: `${info.label} — ${info.host}`,
        sections: [
          { label: 'Endpoints', items: info.endpoints },
          { label: 'Holds', items: info.holds },
        ],
      }
    },
    [tokenIssued, resourceFetched],
  )

  return (
    <>
      <header>
        <h1>OAuth 2.0 — Client Credentials Grant</h1>
        <p className="subtitle">RFC 6749 §4.4 — machine-to-machine, no user, no redirect. Pick an outcome at each branch.</p>
      </header>

      <FlowCanvas
        actorMeta={ACTOR_META}
        edgePairs={EDGE_PAIRS}
        from={step.from}
        to={step.to}
        onActorClick={onActorClick}
        pulseKey={pulseKey}
      />

      <ActorInfoPanel openActor={state.openActor} getInfo={getInfo} onClose={() => dispatch({ type: 'CLOSE_ACTOR' })} />

      <StepPanel index={state.cursor} total={state.path.length} outcome={outcome} />

      <OutcomePicker
        outcomes={step.outcomes}
        selectedId={outcome.id}
        onSelect={(outcomeId) => dispatch({ type: 'CHOOSE_OUTCOME', outcomeId })}
      />

      <PlaybackControls
        atStart={atStart}
        atEnd={atEnd}
        playing={state.playing}
        onReset={() => dispatch({ type: 'RESET' })}
        onPrev={() => dispatch({ type: 'PREV' })}
        onNext={() => dispatch({ type: 'NEXT' })}
        onTogglePlay={() => dispatch({ type: 'TOGGLE_PLAY' })}
      />
    </>
  )
}
