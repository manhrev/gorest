import type { ActorId } from '../types'
import { actorInfo, clientHolding } from '../flowGraph'

interface Props {
  openActor: ActorId | null
  tokenIssued: boolean
  resourceFetched: boolean
  onClose: () => void
}

export default function ActorInfoPanel({ openActor, tokenIssued, resourceFetched, onClose }: Props) {
  if (!openActor) return null

  return (
    <div className="info-panel">
      <button className="info-close" onClick={onClose}>
        ✕
      </button>
      {openActor === 'client' ? (
        <>
          <h3>Client — holding</h3>
          <ul>
            {clientHolding({ tokenIssued, resourceFetched }).map((x) => (
              <li key={x}>{x}</li>
            ))}
          </ul>
        </>
      ) : (
        (() => {
          const info = actorInfo[openActor]
          return (
            <>
              <h3>
                {info.label} — {info.host}
              </h3>
              <p className="info-section-label">Endpoints</p>
              <ul>
                {info.endpoints.map((x) => (
                  <li key={x}>{x}</li>
                ))}
              </ul>
              <p className="info-section-label">Holds</p>
              <ul>
                {info.holds.map((x) => (
                  <li key={x}>{x}</li>
                ))}
              </ul>
            </>
          )
        })()
      )}
    </div>
  )
}
