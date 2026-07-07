// Thin helpers over chess.js shared by the board component and the
// creation/review flows.
import { Chess, validateFen } from 'chess.js'
import type { Square } from 'chess.js'

export type Color = 'white' | 'black'

export function fenIsValid(fen: string): boolean {
  return validateFen(fen.trim()).ok
}

export function turnOf(fen: string): Color {
  return new Chess(fen).turn() === 'w' ? 'white' : 'black'
}

export function inCheck(fen: string): boolean {
  return new Chess(fen).inCheck()
}

/** Legal destination map in the shape chessground expects. */
export function legalDests(fen: string): Map<Square, Square[]> {
  const chess = new Chess(fen)
  const dests = new Map<Square, Square[]>()
  for (const m of chess.moves({ verbose: true })) {
    const arr = dests.get(m.from) ?? []
    arr.push(m.to)
    dests.set(m.from, arr)
  }
  return dests
}

/** Apply a SAN move; returns the resulting FEN. Throws on illegal moves. */
export function applySan(fen: string, san: string): string {
  const chess = new Chess(fen)
  chess.move(san)
  return chess.fen()
}

/** Apply a whole SAN line; returns each intermediate FEN (starting FEN first). */
export function fensOfLine(fen: string, line: string[]): string[] {
  const chess = new Chess(fen)
  const fens = [chess.fen()]
  for (const san of line) {
    chess.move(san)
    fens.push(chess.fen())
  }
  return fens
}

/** from/to squares of a SAN move in a position (for drawing arrows). */
export function sanToSquares(fen: string, san: string): { from: Square; to: Square } {
  const chess = new Chess(fen)
  const m = chess.move(san)
  return { from: m.from, to: m.to }
}

/**
 * Resolve a user's from/to (+optional promotion) input to a legal move.
 * Returns the SAN and resulting FEN, or null if the move is not legal.
 */
export function tryMove(
  fen: string,
  from: string,
  to: string,
  promotion?: string,
): { san: string; fen: string } | null {
  const chess = new Chess(fen)
  try {
    const m = chess.move({ from, to, promotion })
    return { san: m.san, fen: chess.fen() }
  } catch {
    return null
  }
}

/** True if from→to is a promotion move in this position. */
export function isPromotion(fen: string, from: string, to: string): boolean {
  const chess = new Chess(fen)
  return chess
    .moves({ verbose: true })
    .some((m) => m.from === from && m.to === to && m.promotion !== undefined)
}

/** SAN equality ignoring check/mate/annotation suffixes ("Qxf7#" ≡ "Qxf7"). */
export function sameSan(a: string, b: string): boolean {
  const norm = (s: string) => s.replace(/[+#?!]+$/g, '')
  return norm(a.trim()) === norm(b.trim())
}

/** Format a SAN line with move numbers from a starting FEN, e.g. "12…Nf6 13.Bg5". */
export function numberedLine(fen: string, line: string[]): string {
  const chess = new Chess(fen)
  let moveNo = parseInt(fen.split(' ')[5] ?? '1', 10) || 1
  let turn = chess.turn()
  const parts: string[] = []
  for (const san of line) {
    if (turn === 'w') parts.push(`${moveNo}.${san}`)
    else {
      parts.push(parts.length === 0 ? `${moveNo}…${san}` : san)
      moveNo++
    }
    turn = turn === 'w' ? 'b' : 'w'
  }
  return parts.join(' ')
}

export const START_FEN = new Chess().fen()
