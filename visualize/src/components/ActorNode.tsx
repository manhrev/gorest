import type { NodeProps } from 'reactflow'
import type { ActorId } from '../types'

export interface ActorNodeData {
  actor: ActorId
  icon: string
  label: string
  active: boolean
  onClick: (actor: ActorId) => void
}

export default function ActorNode({ data }: NodeProps<ActorNodeData>) {
  return (
    <div
      className={`actor-node${data.active ? ' active' : ''}`}
      data-actor={data.actor}
      tabIndex={0}
      onClick={() => data.onClick(data.actor)}
    >
      <div className="actor-icon">{data.icon}</div>
      <div className="actor-label">{data.label}</div>
    </div>
  )
}
