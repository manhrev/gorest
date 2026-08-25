import { useEffect, useMemo, useRef } from 'react'
import ReactFlow, { type Node } from 'reactflow'
import 'reactflow/dist/style.css'
import ActorNode, { type ActorNodeData } from './ActorNode'
import type { ActorId } from '../types'

const nodeTypes = { actor: ActorNode }

export interface ActorMeta {
  icon: string
  label: string
  position: { x: number; y: number }
}

interface Props {
  actorMeta: Record<string, ActorMeta>
  /** static dashed lines to always draw, e.g. [['client','auth'], ['client','resource']] */
  edgePairs: [ActorId, ActorId][]
  from: ActorId | null
  to: ActorId | null
  onActorClick: (actor: ActorId) => void
  /** bumped whenever the active step/outcome changes, to retrigger the pulse */
  pulseKey: string
}

export default function FlowCanvas({ actorMeta, edgePairs, from, to, onActorClick, pulseKey }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const actors = useMemo(() => Object.keys(actorMeta), [actorMeta])

  const nodes: Node<ActorNodeData>[] = useMemo(
    () =>
      actors.map((actor) => ({
        id: actor,
        type: 'actor',
        position: actorMeta[actor].position,
        data: {
          actor,
          icon: actorMeta[actor].icon,
          label: actorMeta[actor].label,
          active: actor === from || actor === to,
          onClick: onActorClick,
        },
        draggable: false,
        connectable: false,
      })),
    [actors, actorMeta, from, to, onActorClick],
  )

  // Static dashed lines + animated pulse, drawn as a plain SVG overlay on top of the
  // React Flow canvas — ported ~verbatim from the original vanilla-JS implementation,
  // since RF's own edge system fights the fixed-layout diagrams we want here.
  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    const svg = container.querySelector<SVGSVGElement>('.flow-overlay')
    if (!svg) return

    const center = (actor: ActorId) => {
      const el = container.querySelector<HTMLElement>(`.actor-node[data-actor="${actor}"]`)
      if (!el) return { x: 0, y: 0 }
      const box = el.getBoundingClientRect()
      const stageBox = container.getBoundingClientRect()
      return { x: box.left + box.width / 2 - stageBox.left, y: box.top + box.height / 2 - stageBox.top }
    }

    const draw = () => {
      svg.setAttribute('viewBox', `0 0 ${container.clientWidth} ${container.clientHeight}`)
      svg.innerHTML = ''

      for (const [a, b] of edgePairs) {
        const p1 = center(a)
        const p2 = center(b)
        const line = document.createElementNS('http://www.w3.org/2000/svg', 'line')
        line.setAttribute('x1', String(p1.x))
        line.setAttribute('y1', String(p1.y))
        line.setAttribute('x2', String(p2.x))
        line.setAttribute('y2', String(p2.y))
        line.setAttribute('class', 'arrow-line')
        svg.appendChild(line)
      }

      if (from && to && from !== to) {
        const p1 = center(from)
        const p2 = center(to)
        const dot = document.createElementNS('http://www.w3.org/2000/svg', 'circle')
        dot.setAttribute('r', '6')
        dot.setAttribute('cx', String(p1.x))
        dot.setAttribute('cy', String(p1.y))
        dot.setAttribute('class', 'pulse')
        svg.appendChild(dot)
        dot.animate(
          [
            { cx: p1.x, cy: p1.y, opacity: 0 },
            { cx: p1.x, cy: p1.y, opacity: 1, offset: 0.15 },
            { cx: p2.x, cy: p2.y, opacity: 1 },
          ],
          { duration: 900, easing: 'ease-in-out', fill: 'forwards' },
        )
      }
    }

    draw()
    window.addEventListener('resize', draw)
    return () => window.removeEventListener('resize', draw)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, to, pulseKey, edgePairs])

  return (
    <div className="stage" ref={containerRef}>
      <svg className="flow-overlay arrows" />
      <ReactFlow
        nodes={nodes}
        edges={[]}
        nodeTypes={nodeTypes}
        nodesDraggable={false}
        nodesConnectable={false}
        panOnDrag={false}
        zoomOnScroll={false}
        zoomOnPinch={false}
        zoomOnDoubleClick={false}
        preventScrolling={false}
        proOptions={{ hideAttribution: true }}
        minZoom={0.5}
        maxZoom={1.6}
        fitView
        fitViewOptions={{ padding: 0.15, maxZoom: 1.6 }}
      />
    </div>
  )
}
