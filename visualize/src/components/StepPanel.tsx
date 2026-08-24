import type { Outcome } from '../types'

interface Props {
  index: number
  total: number
  outcome: Outcome
}

export default function StepPanel({ index, total, outcome }: Props) {
  return (
    <div className="panel">
      <div className="panel-head">
        <span className="step-count">
          Step {index + 1} / {total}
        </span>
        <h2>{outcome.title}</h2>
      </div>
      <p id="step-detail">{outcome.detail}</p>
      <pre id="step-http">{outcome.http}</pre>
    </div>
  )
}
