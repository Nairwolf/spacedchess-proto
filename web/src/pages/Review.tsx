// Review session (DESIGN.md §4.1): dominant board, quiet side panel, thin
// header, grading/result strip only after an attempt or reveal.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import type { BlunderDetails, Card, StrategicDetails, TacticalDetails } from '../api'
import Board from '../components/Board'
import { sideName, typeLabel } from '../cardText'
import { applySan, fensOfLine, numberedLine, sameSan, sanToSquares } from '../chessUtil'
import type { Color } from '../chessUtil'

type AttemptStatus = 'attempt' | 'ok' | 'fail'

/**
 * One attempt at a SAN line where the user plays the moves at even indices
 * and the odd-index replies are auto-played. Used by tactical cards and by
 * both blunder steps.
 */
function useLineAttempt(baseFen: string, line: string[], onDone: (ok: boolean) => void) {
  const [fen, setFen] = useState(baseFen)
  const [moveIdx, setMoveIdx] = useState(0)
  const [status, setStatus] = useState<AttemptStatus>('attempt')
  const [lastMove, setLastMove] = useState<[string, string] | undefined>()
  const timer = useRef<number | undefined>(undefined)

  useEffect(() => () => window.clearTimeout(timer.current), [])

  const playSan = useCallback((fromFen: string, san: string): string => {
    const sq = sanToSquares(fromFen, san)
    setLastMove([sq.from, sq.to])
    const next = applySan(fromFen, san)
    setFen(next)
    return next
  }, [])

  const onUserMove = useCallback(
    (san: string) => {
      if (status !== 'attempt') return
      if (!sameSan(san, line[moveIdx])) {
        // Show the move they actually played, then mark the attempt failed.
        try {
          playSan(fen, san)
        } catch {
          /* board already validated legality */
        }
        setStatus('fail')
        onDone(false)
        return
      }
      const afterUser = playSan(fen, line[moveIdx])
      if (moveIdx + 1 >= line.length) {
        setStatus('ok')
        onDone(true)
        return
      }
      // Auto-play the forced reply after a beat.
      timer.current = window.setTimeout(() => {
        playSan(afterUser, line[moveIdx + 1])
        if (moveIdx + 2 >= line.length) {
          setStatus('ok')
          onDone(true)
        } else {
          setMoveIdx(moveIdx + 2)
        }
      }, 300)
      setMoveIdx(moveIdx + 1) // block input while the reply plays
    },
    [status, line, moveIdx, fen, onDone, playSan],
  )

  // After resolution: step through the correct line move by move.
  const lineFens = useMemo(() => {
    try {
      return fensOfLine(baseFen, line)
    } catch {
      return [baseFen]
    }
  }, [baseFen, line])
  const [replayIdx, setReplayIdx] = useState<number | null>(null)

  const replay = useCallback(
    (dir: 1 | -1) => {
      setReplayIdx((cur) => {
        const start = cur === null ? (dir === 1 ? 0 : lineFens.length - 1) : cur
        const next = Math.min(Math.max(cur === null ? start : cur + dir, 0), lineFens.length - 1)
        setFen(lineFens[next])
        if (next > 0) {
          try {
            const sq = sanToSquares(lineFens[next - 1], line[next - 1])
            setLastMove([sq.from, sq.to])
          } catch {
            setLastMove(undefined)
          }
        } else {
          setLastMove(undefined)
        }
        return next
      })
    },
    [lineFens, line],
  )

  const inputTurn = status === 'attempt' && moveIdx % 2 === 0
  return { fen, status, onUserMove, interactive: inputTurn, lastMove, replay, replayIdx }
}

interface FlowProps {
  card: Card
  onResolved: (correct: boolean) => void
  onNext: () => void
}

function ReplayControls({
  replay,
  label,
}: {
  replay: (dir: 1 | -1) => void
  label: string
}) {
  return (
    <span className="review__replay">
      <span className="review__verdict-line mono">{label}</span>{' '}
      <button type="button" className="btn btn--sm" onClick={() => replay(-1)} aria-label="Previous move">
        ‹
      </button>{' '}
      <button type="button" className="btn btn--sm" onClick={() => replay(1)} aria-label="Next move">
        ›
      </button>
    </span>
  )
}

function TacticalFlow({ card, onResolved, onNext }: FlowProps) {
  const details = card.details as TacticalDetails
  const attempt = useLineAttempt(card.fen, details.solution, onResolved)
  const userSide = sideName(card.side_to_move)
  const resolved = attempt.status !== 'attempt'

  return (
    <ReviewScreenBody
      card={card}
      boardFen={attempt.fen}
      orientation={card.side_to_move === 'w' ? 'white' : 'black'}
      interactive={attempt.interactive}
      onUserMove={attempt.onUserMove}
      lastMove={attempt.lastMove}
      panel={
        <>
          <p>
            {userSide} to move. Find the best {details.solution.length > 1 ? 'line' : 'move'}.
          </p>
          {resolved && (
            <p className="mono reveal">{numberedLine(card.fen, details.solution)}</p>
          )}
        </>
      }
      footer={
        resolved && (
          <>
            <span
              className={`review__verdict review__verdict--${attempt.status === 'ok' ? 'correct' : 'incorrect'}`}
            >
              {attempt.status === 'ok' ? 'Correct.' : 'Incorrect.'}
            </span>
            <ReplayControls replay={attempt.replay} label="Step through the solution" />
            <button type="button" className="btn btn--primary" onClick={onNext} autoFocus>
              Next
            </button>
          </>
        )
      }
    />
  )
}

function BlunderFlow({ card, onResolved, onNext }: FlowProps) {
  const details = card.details as BlunderDetails
  const [phase, setPhase] = useState<'preview' | 'step1' | 'between' | 'step2' | 'done'>('preview')
  const [step1OK, setStep1OK] = useState<boolean | null>(null)
  const [step2OK, setStep2OK] = useState<boolean | null>(null)

  const fenAfterBlunder = useMemo(
    () => applySan(card.fen, details.intended_move),
    [card.fen, details.intended_move],
  )
  const opponent = sideName(card.side_to_move === 'w' ? 'b' : 'w')
  const userSide = sideName(card.side_to_move)

  const step1 = useLineAttempt(fenAfterBlunder, details.refutation, (ok) => {
    setStep1OK(ok)
    setPhase('between')
  })
  const step2 = useLineAttempt(card.fen, details.correct_alternative, (ok) => {
    setStep2OK(ok)
    setPhase('done')
    onResolved(step1OK === true && ok)
  })

  const intendedArrow = useMemo(() => {
    try {
      const sq = sanToSquares(card.fen, details.intended_move)
      return [{ orig: sq.from, dest: sq.to, brush: 'red' }]
    } catch {
      return []
    }
  }, [card.fen, details.intended_move])

  const boardProps =
    phase === 'preview'
      ? { fen: card.fen, interactive: false, shapes: intendedArrow }
      : phase === 'step1' || phase === 'between'
        ? {
            fen: step1.fen,
            interactive: phase === 'step1' && step1.interactive,
            onUserMove: step1.onUserMove,
            lastMove: step1.lastMove,
          }
        : {
            fen: step2.fen,
            interactive: phase === 'step2' && step2.interactive,
            onUserMove: step2.onUserMove,
            lastMove: step2.lastMove,
          }

  const stepClass = (ok: boolean | null, active: boolean) =>
    ok === true ? 'step--ok' : ok === false ? 'step--fail' : active ? 'step--active' : ''

  return (
    <ReviewScreenBody
      card={card}
      boardFen={boardProps.fen}
      orientation={card.side_to_move === 'w' ? 'white' : 'black'}
      interactive={boardProps.interactive ?? false}
      onUserMove={boardProps.onUserMove}
      lastMove={boardProps.lastMove}
      shapes={boardProps.shapes}
      panel={
        <>
          <p className="review__intended">
            You're about to play: <span className="mono">{details.intended_move}</span>
          </p>
          <div className="review__steps">
            <span className={stepClass(step1OK, phase === 'step1' || phase === 'preview')}>
              1. Find {opponent}'s refutation
              {step1OK !== null && (step1OK ? ' — found' : ' — missed')}
            </span>
            <span className={stepClass(step2OK, phase === 'step2')}>
              2. Find what {userSide} should play instead
              {step2OK !== null && (step2OK ? ' — found' : ' — missed')}
            </span>
          </div>

          {phase === 'preview' && (
            <button type="button" className="btn btn--primary" onClick={() => setPhase('step1')}>
              Play {details.intended_move} and find the refutation
            </button>
          )}

          {(phase === 'between' || phase === 'done') && (
            <p className="mono reveal">
              {numberedLine(fenAfterBlunder, details.refutation)}
            </p>
          )}
          {phase === 'done' && (
            <p className="mono reveal">
              Instead: {numberedLine(card.fen, details.correct_alternative)}
            </p>
          )}
        </>
      }
      footer={
        phase === 'between' ? (
          <>
            <span
              className={`review__verdict review__verdict--${step1OK ? 'correct' : 'incorrect'}`}
            >
              {step1OK ? 'Refutation found.' : `Missed — the refutation was ${details.refutation[0]}.`}
            </span>
            <ReplayControls replay={step1.replay} label="Step through it" />
            <button type="button" className="btn btn--primary" onClick={() => setPhase('step2')} autoFocus>
              Now find the better move
            </button>
          </>
        ) : phase === 'done' ? (
          <>
            <span
              className={`review__verdict review__verdict--${step1OK && step2OK ? 'correct' : 'incorrect'}`}
            >
              {step1OK && step2OK ? 'Correct — both steps.' : 'Incorrect.'}
            </span>
            <ReplayControls replay={step2.replay} label="Step through the alternative" />
            <button type="button" className="btn btn--primary" onClick={onNext} autoFocus>
              Next
            </button>
          </>
        ) : null
      }
    />
  )
}

function StrategicFlow({ card, onResolved, onNext }: FlowProps) {
  const details = card.details as StrategicDetails
  const [revealed, setRevealed] = useState(false)

  const grade = (remembered: boolean) => {
    onResolved(remembered)
    onNext()
  }

  return (
    <ReviewScreenBody
      card={card}
      boardFen={card.fen}
      orientation={card.side_to_move === 'w' ? 'white' : 'black'}
      interactive={false}
      panel={
        <>
          <p className="review__prompt-text">{details.question}</p>
          {!revealed ? (
            <button type="button" className="btn btn--primary" onClick={() => setRevealed(true)}>
              Show answer
            </button>
          ) : (
            <div className="review__answer reveal">{details.answer}</div>
          )}
        </>
      }
      footer={
        revealed && (
          <>
            <button type="button" className="btn" onClick={() => grade(false)}>
              Didn't remember
            </button>
            <button type="button" className="btn btn--primary" onClick={() => grade(true)} autoFocus>
              Remembered
            </button>
          </>
        )
      }
    />
  )
}

interface BodyProps {
  card: Card
  boardFen: string
  orientation: Color
  interactive: boolean
  onUserMove?: (san: string) => void
  lastMove?: [string, string]
  shapes?: { orig: string; dest: string; brush: string }[]
  panel: React.ReactNode
  footer: React.ReactNode
}

function ReviewScreenBody({
  card,
  boardFen,
  orientation,
  interactive,
  onUserMove,
  lastMove,
  shapes,
  panel,
  footer,
}: BodyProps) {
  return (
    <>
      <div className="review__body">
        <div className="review__board">
          <Board
            fen={boardFen}
            orientation={orientation}
            interactive={interactive}
            onUserMove={onUserMove}
            lastMove={lastMove}
            shapes={shapes as never}
          />
        </div>
        <div className="review__panel">
          <div className="card-type-label">{typeLabel(card.card_type)}</div>
          <div className="review__prompt">{panel}</div>
        </div>
      </div>
      <div className="review__footer">{footer}</div>
    </>
  )
}

export default function Review() {
  const [params] = useSearchParams()
  const setID = params.get('set_id')
  const tag = params.get('tag')

  const [queue, setQueue] = useState<Card[] | null>(null)
  const [idx, setIdx] = useState(0)
  const [correctCount, setCorrectCount] = useState(0)
  const [scopeName, setScopeName] = useState('')
  const [error, setError] = useState('')
  const resolvedRef = useRef(false)

  useEffect(() => {
    api
      .dueCards({ set_id: setID ? Number(setID) : undefined, tag: tag ?? undefined })
      .then(setQueue)
      .catch((e) => setError(e.message))
    if (setID) {
      api
        .listSets()
        .then((sets) => {
          const s = sets.find((x) => x.id === Number(setID))
          if (s) setScopeName(`Set: ${s.name}`)
        })
        .catch(() => {})
    } else if (tag) {
      setScopeName(`Tag: ${tag}`)
    }
  }, [setID, tag])

  const card = queue?.[idx]

  const onResolved = useCallback(
    (correct: boolean) => {
      if (!card || resolvedRef.current) return
      resolvedRef.current = true
      if (correct) setCorrectCount((n) => n + 1)
      api.submitReview(card.id, correct).catch((e) => setError(e.message))
    },
    [card],
  )

  const onNext = useCallback(() => {
    resolvedRef.current = false
    setIdx((i) => i + 1)
  }, [])

  return (
    <div className="review">
      <header className="review__header">
        <Link to="/">← Library</Link>
        <span>{scopeName}</span>
        <span className="review__progress">
          {queue && queue.length > 0 && idx < queue.length ? `${idx + 1} / ${queue.length} due` : ''}
        </span>
      </header>

      {error && <p className="form-error" style={{ padding: '0 14px' }}>{error}</p>}

      {queue === null ? null : queue.length === 0 ? (
        <div className="review__done">
          <strong>Nothing due.</strong>
          <p>Come back when cards are scheduled, or add new ones.</p>
          <Link to="/" className="btn">
            Back to library
          </Link>
        </div>
      ) : idx >= queue.length ? (
        <div className="review__done">
          <strong>Session complete.</strong>
          <p>
            {correctCount} of {queue.length} correct.
          </p>
          <Link to="/" className="btn">
            Back to library
          </Link>
        </div>
      ) : (
        card && (
          <CardFlow key={card.id} card={card} onResolved={onResolved} onNext={onNext} />
        )
      )}
    </div>
  )
}

function CardFlow(props: FlowProps) {
  switch (props.card.card_type) {
    case 'tactical_opportunity':
      return <TacticalFlow {...props} />
    case 'blunder':
      return <BlunderFlow {...props} />
    case 'strategic_mistake':
      return <StrategicFlow {...props} />
  }
}
