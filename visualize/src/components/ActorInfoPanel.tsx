import type { ActorId } from '../types'

export interface InfoSection {
  /** omit/empty to render a plain list with no section heading */
  label?: string
  items: string[]
}

export interface ActorInfoContent {
  title: string
  sections: InfoSection[]
}

interface Props {
  openActor: ActorId | null
  getInfo: (actor: ActorId) => ActorInfoContent
  onClose: () => void
}

export default function ActorInfoPanel({ openActor, getInfo, onClose }: Props) {
  if (!openActor) return null
  const info = getInfo(openActor)

  return (
    <div className="info-panel">
      <button className="info-close" onClick={onClose}>
        ✕
      </button>
      <h3>{info.title}</h3>
      {info.sections.map((section, i) => (
        <div key={i}>
          {section.label && <p className="info-section-label">{section.label}</p>}
          <ul>
            {section.items.map((x) => (
              <li key={x}>{x}</li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  )
}
