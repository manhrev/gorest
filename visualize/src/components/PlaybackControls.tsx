interface Props {
  atStart: boolean
  atEnd: boolean
  playing: boolean
  onReset: () => void
  onPrev: () => void
  onNext: () => void
  onTogglePlay: () => void
}

export default function PlaybackControls({ atStart, atEnd, playing, onReset, onPrev, onNext, onTogglePlay }: Props) {
  return (
    <div className="controls">
      <button onClick={onReset}>↺ Reset</button>
      <button onClick={onPrev} disabled={atStart}>
        ◀ Prev
      </button>
      <button onClick={onTogglePlay} disabled={atEnd && !playing}>
        {playing ? '⏸ Pause' : '▶ Play'}
      </button>
      <button onClick={onNext} disabled={atEnd}>
        Next ▶
      </button>
    </div>
  )
}
