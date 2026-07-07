// Shared chessground wrapper used by creation, review, and browse screens.
// Legal moves come from chess.js; illegal moves are impossible to input.
import { useEffect, useRef, useState } from 'react'
import { Chessground } from 'chessground'
import type { Api } from 'chessground/api'
import type { Config } from 'chessground/config'
import type { Key } from 'chessground/types'
import type { DrawShape } from 'chessground/draw'
import { inCheck, isPromotion, legalDests, turnOf, tryMove } from '../chessUtil'
import type { Color } from '../chessUtil'

export interface BoardProps {
  fen: string
  orientation: Color
  /** When true, the side to move can input moves. */
  interactive?: boolean
  /** Called with the SAN of a legal move the user entered. */
  onUserMove?: (san: string) => void
  /** Arrows/highlights (e.g. the intended blunder move). */
  shapes?: DrawShape[]
  lastMove?: [string, string]
  /** Small non-interactive rendering for library rows. */
  mini?: boolean
}

const PROMO_PIECES = [
  { code: 'q', label: 'Queen' },
  { code: 'r', label: 'Rook' },
  { code: 'b', label: 'Bishop' },
  { code: 'n', label: 'Knight' },
] as const

export default function Board(props: BoardProps) {
  const { fen, orientation, interactive = false, onUserMove, shapes, lastMove, mini = false } = props
  const elRef = useRef<HTMLDivElement>(null)
  const apiRef = useRef<Api | null>(null)
  const [promo, setPromo] = useState<{ from: Key; to: Key } | null>(null)

  // Kept in refs so the chessground move callback always sees current values.
  const stateRef = useRef({ fen, onUserMove })
  stateRef.current = { fen, onUserMove }

  const buildConfig = (): Config => {
    const cfg: Config = {
      fen,
      orientation,
      turnColor: turnOf(fen),
      coordinates: !mini,
      viewOnly: mini,
      check: inCheck(fen),
      animation: {
        enabled: !window.matchMedia('(prefers-reduced-motion: reduce)').matches && !mini,
        duration: 180,
      },
      lastMove: lastMove as Key[] | undefined,
      draggable: { enabled: interactive, showGhost: true },
      selectable: { enabled: interactive },
      movable: {
        free: false,
        color: interactive ? turnOf(fen) : undefined,
        dests: interactive ? (legalDests(fen) as Map<Key, Key[]>) : new Map(),
        showDests: true,
        events: {
          after: (orig: Key, dest: Key) => {
            const { fen: cur, onUserMove: cb } = stateRef.current
            if (!cb) return
            if (isPromotion(cur, orig, dest)) {
              setPromo({ from: orig, to: dest })
              return
            }
            const res = tryMove(cur, orig, dest)
            if (res) cb(res.san)
          },
        },
      },
      drawable: {
        enabled: false,
        visible: true,
        autoShapes: shapes ?? [],
      },
    }
    return cfg
  }

  useEffect(() => {
    if (!elRef.current) return
    apiRef.current = Chessground(elRef.current, buildConfig())
    const onResize = () => apiRef.current?.redrawAll()
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
      apiRef.current?.destroy()
      apiRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    apiRef.current?.set(buildConfig())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fen, orientation, interactive, mini, JSON.stringify(shapes), JSON.stringify(lastMove)])

  const choosePromotion = (code: string) => {
    if (!promo) return
    const { fen: cur, onUserMove: cb } = stateRef.current
    const res = tryMove(cur, promo.from, promo.to, code)
    setPromo(null)
    if (res && cb) cb(res.san)
    else apiRef.current?.set({ fen: cur }) // snap back if somehow illegal
  }

  const cancelPromotion = () => {
    setPromo(null)
    apiRef.current?.set({ fen: stateRef.current.fen })
  }

  return (
    <div className={mini ? 'board board--mini' : 'board'}>
      <div ref={elRef} className="board__cg" />
      {promo && (
        <div className="board__promo" role="dialog" aria-label="Choose promotion piece">
          {PROMO_PIECES.map((p) => (
            <button key={p.code} type="button" onClick={() => choosePromotion(p.code)}>
              {p.label}
            </button>
          ))}
          <button type="button" className="board__promo-cancel" onClick={cancelPromotion}>
            Cancel
          </button>
        </div>
      )}
    </div>
  )
}
