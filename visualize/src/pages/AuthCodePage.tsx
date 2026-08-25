import { useCallback, useEffect, useMemo, useReducer, useState } from 'react'
import FlowCanvas, { type ActorMeta } from '../components/FlowCanvas'
import StepPanel from '../components/StepPanel'
import OutcomePicker from '../components/OutcomePicker'
import ActorInfoPanel, { type ActorInfoContent } from '../components/ActorInfoPanel'
import PlaybackControls from '../components/PlaybackControls'
import { buildGraph, actorInfo, clientHolding, userHolding } from './authCodeGraph'
import { makeReducer, initState } from '../reducer'
import type { ActorId } from '../types'

const PLAY_INTERVAL_MS = 1800

const ACTOR_META: Record<string, ActorMeta> = {
  user: { icon: '🌐', label: 'User (Browser)', position: { x: 230, y: 0 } },
  auth: { icon: '🔐', label: 'Auth Server', position: { x: 0, y: 150 } },
  resource: { icon: '📦', label: 'Resource Server', position: { x: 460, y: 150 } },
  client: { icon: '🖧', label: 'OAuth Client', position: { x: 230, y: 300 } },
}

const EDGE_PAIRS: [ActorId, ActorId][] = [
  ['user', 'auth'],
  ['user', 'client'],
  ['client', 'auth'],
  ['client', 'resource'],
]

export default function AuthCodePage() {
  const [usePkce, setUsePkce] = useState(true)
  const graph = useMemo(() => buildGraph(usePkce), [usePkce])
  const reducer = useMemo(() => makeReducer(graph), [graph])
  const [state, dispatch] = useReducer(reducer, undefined, initState)

  // outcomes differ per PKCE setting (an extra error branch) — start over on toggle
  // rather than risk pointing at an outcome that no longer exists in the new graph
  useEffect(() => {
    dispatch({ type: 'RESET' })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [usePkce])

  const entry = state.path[state.cursor]
  const step = graph[entry.stepId]
  const outcome = step.outcomes.find((o) => o.id === entry.outcomeId)!
  const atStart = state.cursor === 0
  const atEnd = outcome.next === null

  const visited = state.path.slice(0, state.cursor + 1)
  const pkceGenerated = visited.some((p) => p.stepId === 'redirect-to-auth')
  const stateIssued = pkceGenerated
  const codeReceived = visited.some((p) => p.stepId === 'follow-redirect-client')
  const tokenIssued = visited.some((p) => p.stepId === 'issue-tokens')
  const resourceFetched = visited.some((p) => p.stepId === 'resource-returned')

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
        return {
          title: 'Client — holding',
          sections: [{ items: clientHolding({ pkceGenerated, codeReceived, tokenIssued, resourceFetched }, usePkce) }],
        }
      }
      if (actor === 'user') {
        return { title: 'User (Browser) — holding', sections: [{ items: userHolding({ stateIssued }) }] }
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
    [pkceGenerated, codeReceived, tokenIssued, resourceFetched, stateIssued, usePkce],
  )

  return (
    <>
      <header>
        <h1>OAuth 2.0 — Authorization Code Grant</h1>
        <p className="subtitle">
          RFC 6749 §4.1 — confidential client (backend), user goes through login + consent.{' '}
          <label className="pkce-toggle">
            <input type="checkbox" checked={usePkce} onChange={(e) => setUsePkce(e.target.checked)} />
            Use PKCE (RFC 7636)
          </label>
        </p>
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
