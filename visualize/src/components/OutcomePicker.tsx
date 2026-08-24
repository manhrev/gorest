import type { Outcome } from '../types'

interface Props {
  outcomes: Outcome[]
  selectedId: string
  onSelect: (outcomeId: string) => void
}

export default function OutcomePicker({ outcomes, selectedId, onSelect }: Props) {
  // always mount this row, even for single-outcome steps, so the controls
  // below don't shift up/down as the branch picker appears/disappears
  if (outcomes.length <= 1) return <div className="outcome-picker outcome-picker-empty" aria-hidden="true" />

  return (
    <div className="outcome-picker" role="radiogroup" aria-label="Choose outcome">
      {outcomes.map((o) => (
        <button
          key={o.id}
          type="button"
          role="radio"
          aria-checked={o.id === selectedId}
          className={`outcome-btn outcome-${o.kind}${o.id === selectedId ? ' selected' : ''}`}
          onClick={() => onSelect(o.id)}
        >
          {o.label}
          {o.kind === 'happy' && <span className="outcome-default"> (default)</span>}
        </button>
      ))}
    </div>
  )
}
