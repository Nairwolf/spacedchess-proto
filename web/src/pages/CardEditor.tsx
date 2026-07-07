// Card creation/editing: position setup on one side, type-specific
// authoring on the other (DESIGN.md §4.3). Solutions are recorded by
// playing moves on the board, so only legal lines can be authored.
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import type { BlunderDetails, CardPayload, CardSet, CardType, StrategicDetails, TacticalDetails } from '../api'
import Board from '../components/Board'
import TagInput from '../components/TagInput'
import { START_FEN, fenIsValid, fensOfLine, numberedLine, turnOf } from '../chessUtil'
import { sideName } from '../cardText'

type RecordTarget = 'solution' | 'intended' | 'refutation' | 'alternative'

interface LineFieldProps {
  label: string
  hint?: string
  moves: string[]
  numbered: string
  recording: boolean
  disabled?: boolean
  onRecord: () => void
  onUndo: () => void
}

function LineField({ label, hint, moves, numbered, recording, disabled, onRecord, onUndo }: LineFieldProps) {
  return (
    <div className="line-field">
      <div className="line-field__head">
        <span>{label}</span>
        {moves.length > 0 && (
          <button type="button" className="btn btn--ghost btn--sm" onClick={onUndo}>
            Undo move
          </button>
        )}
        <button
          type="button"
          className={`btn btn--sm record-btn${recording ? ' btn--primary' : ''}`}
          disabled={disabled}
          onClick={onRecord}
        >
          {recording ? 'Recording — play moves on the board' : 'Record on board'}
        </button>
      </div>
      <div className={`line-field__moves${recording ? ' recording' : ''}`}>
        {moves.length > 0 ? numbered : <span className="placeholder">{hint ?? 'No moves yet'}</span>}
      </div>
    </div>
  )
}

export default function CardEditor() {
  const { id } = useParams()
  const cardID = id ? Number(id) : null
  const navigate = useNavigate()

  const [cardType, setCardType] = useState<CardType>('tactical_opportunity')
  const [fen, setFen] = useState(START_FEN)
  const [fenInput, setFenInput] = useState(START_FEN)
  const [solution, setSolution] = useState<string[]>([])
  const [intended, setIntended] = useState<string[]>([]) // 0 or 1 move
  const [refutation, setRefutation] = useState<string[]>([])
  const [alternative, setAlternative] = useState<string[]>([])
  const [question, setQuestion] = useState('')
  const [answer, setAnswer] = useState('')
  const [sourceNote, setSourceNote] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const [setIDs, setSetIDs] = useState<number[]>([])
  const [target, setTarget] = useState<RecordTarget | null>(null)

  const [allTags, setAllTags] = useState<string[]>([])
  const [sets, setSets] = useState<CardSet[]>([])
  const [newSetName, setNewSetName] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [loaded, setLoaded] = useState(cardID === null)

  useEffect(() => {
    api.listTags().then((ts) => setAllTags(ts.map((t) => t.name))).catch(() => {})
    api.listSets().then(setSets).catch(() => {})
  }, [])

  useEffect(() => {
    if (cardID === null) return
    api
      .getCard(cardID)
      .then((card) => {
        setCardType(card.card_type)
        setFen(card.fen)
        setFenInput(card.fen)
        setSourceNote(card.source_note)
        setTags(card.tags)
        setSetIDs(card.set_ids)
        if (card.card_type === 'tactical_opportunity') {
          setSolution((card.details as TacticalDetails).solution)
        } else if (card.card_type === 'blunder') {
          const d = card.details as BlunderDetails
          setIntended([d.intended_move])
          setRefutation(d.refutation)
          setAlternative(d.correct_alternative)
        } else {
          const d = card.details as StrategicDetails
          setQuestion(d.question)
          setAnswer(d.answer)
        }
        setLoaded(true)
      })
      .catch((e) => setError(e.message))
  }, [cardID])

  const fenValid = fenIsValid(fen)

  // Base position and accumulated line for the active recording target.
  const activeLine: string[] =
    target === 'solution' ? solution
    : target === 'intended' ? intended
    : target === 'refutation' ? refutation
    : target === 'alternative' ? alternative
    : []

  const baseFen = target === 'refutation' && intended.length === 1
    ? fensOfLine(fen, intended)[1]
    : fen

  let displayFen = baseFen
  if (!fenValid) {
    displayFen = START_FEN
  } else {
    try {
      const fens = fensOfLine(baseFen, activeLine)
      displayFen = fens[fens.length - 1]
    } catch {
      displayFen = baseFen
    }
  }

  const orientation = fenValid ? turnOf(fen) : 'white'
  const interactive =
    fenValid && target !== null && !(target === 'intended' && intended.length >= 1)

  const applyFenInput = (value: string) => {
    setFenInput(value)
    const v = value.trim()
    if (fenIsValid(v) && v !== fen) {
      setFen(v)
      // Recorded lines are position-dependent; a new position resets them.
      setSolution([])
      setIntended([])
      setRefutation([])
      setAlternative([])
      setTarget(null)
    }
  }

  const onUserMove = (san: string) => {
    if (target === 'solution') setSolution((l) => [...l, san])
    else if (target === 'intended') {
      setIntended([san])
      setTarget('refutation') // next authoring step
    } else if (target === 'refutation') setRefutation((l) => [...l, san])
    else if (target === 'alternative') setAlternative((l) => [...l, san])
  }

  const undo = (t: RecordTarget) => {
    const pop = (l: string[]) => l.slice(0, -1)
    if (t === 'solution') setSolution(pop)
    else if (t === 'intended') {
      setIntended([])
      setRefutation([])
      if (target === 'refutation') setTarget('intended')
    } else if (t === 'refutation') setRefutation(pop)
    else setAlternative(pop)
  }

  const record = (t: RecordTarget) => {
    setTarget(target === t ? null : t)
  }

  const toggleSet = (setId: number) => {
    setSetIDs((ids) => (ids.includes(setId) ? ids.filter((i) => i !== setId) : [...ids, setId]))
  }

  const createSet = async () => {
    if (!newSetName.trim()) return
    try {
      const s = await api.createSet(newSetName)
      setSets((prev) => [...prev, s])
      setSetIDs((ids) => [...ids, s.id])
      setNewSetName('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not create set')
    }
  }

  const save = async () => {
    setError('')
    const details =
      cardType === 'tactical_opportunity'
        ? ({ solution } satisfies TacticalDetails)
        : cardType === 'blunder'
          ? ({
              intended_move: intended[0] ?? '',
              refutation,
              correct_alternative: alternative,
            } satisfies BlunderDetails)
          : ({ question, answer } satisfies StrategicDetails)
    const payload: CardPayload = {
      card_type: cardType,
      fen,
      details,
      source_note: sourceNote,
      tags,
      set_ids: setIDs,
    }
    setBusy(true)
    try {
      if (cardID === null) await api.createCard(payload)
      else await api.updateCard(cardID, payload)
      navigate('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save card')
    } finally {
      setBusy(false)
    }
  }

  const deleteCard = async () => {
    if (cardID === null) return
    if (!window.confirm('Delete this card? Its review history goes with it.')) return
    try {
      await api.deleteCard(cardID)
      navigate('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not delete card')
    }
  }

  if (!loaded) return error ? <p className="form-error">{error}</p> : null

  const boardHint = !fenValid
    ? 'Invalid FEN.'
    : target === null
      ? 'Pick a field on the right to record moves.'
      : interactive
        ? `Recording: play ${turnOf(displayFen) === 'white' ? 'White' : 'Black'}'s move on the board.`
        : 'Intended move recorded — undo it to change it.'

  return (
    <div className="editor">
      <div className="editor__board">
        <Board fen={displayFen} orientation={orientation} interactive={interactive} onUserMove={onUserMove} />
        <p className="editor__board-hint">{boardHint}</p>
      </div>

      <div className="editor__form">
        <h1>{cardID === null ? 'New card' : 'Edit card'}</h1>

        <fieldset>
          <legend>Position</legend>
          <div className="fen-row">
            <label>
              FEN
              <input value={fenInput} onChange={(e) => applyFenInput(e.target.value)} spellCheck={false} />
            </label>
            <button type="button" className="btn btn--sm" onClick={() => applyFenInput(START_FEN)}>
              Start position
            </button>
          </div>
          {!fenIsValid(fenInput) && <p className="form-error">This FEN is not a legal position.</p>}
          {fenValid && (
            <p className="editor__board-hint" style={{ margin: 0 }}>
              {sideName(turnOf(fen) === 'white' ? 'w' : 'b')} to move.
            </p>
          )}
        </fieldset>

        <fieldset>
          <legend>Card type</legend>
          <div className="segmented">
            {(
              [
                ['tactical_opportunity', 'Tactical opportunity'],
                ['blunder', 'Blunder'],
                ['strategic_mistake', 'Strategic mistake'],
              ] as [CardType, string][]
            ).map(([t, label]) => (
              <button
                key={t}
                type="button"
                className={cardType === t ? 'active' : ''}
                onClick={() => {
                  setCardType(t)
                  setTarget(null)
                }}
              >
                {label}
              </button>
            ))}
          </div>

          {cardType === 'tactical_opportunity' && (
            <LineField
              label="Accepted solution (move or forced line)"
              hint="Record the winning move — include forced replies if it's a sequence"
              moves={solution}
              numbered={fenValid ? numberedLine(fen, solution) : ''}
              recording={target === 'solution'}
              onRecord={() => record('solution')}
              onUndo={() => undo('solution')}
            />
          )}

          {cardType === 'blunder' && (
            <>
              <LineField
                label="The move you were about to play (the blunder)"
                hint="Record the single move you played in the game"
                moves={intended}
                numbered={fenValid ? numberedLine(fen, intended) : ''}
                recording={target === 'intended'}
                onRecord={() => record('intended')}
                onUndo={() => undo('intended')}
              />
              <LineField
                label="Opponent's refutation (from after the blunder)"
                hint={intended.length === 0 ? 'Record the blunder first' : 'Record the punishing line'}
                moves={refutation}
                numbered={
                  fenValid && intended.length === 1
                    ? numberedLine(fensOfLine(fen, intended)[1], refutation)
                    : ''
                }
                recording={target === 'refutation'}
                disabled={intended.length === 0}
                onRecord={() => record('refutation')}
                onUndo={() => undo('refutation')}
              />
              <LineField
                label="What you should have played instead"
                hint="Record the better move or line from the original position"
                moves={alternative}
                numbered={fenValid ? numberedLine(fen, alternative) : ''}
                recording={target === 'alternative'}
                onRecord={() => record('alternative')}
                onUndo={() => undo('alternative')}
              />
            </>
          )}

          {cardType === 'strategic_mistake' && (
            <>
              <label>
                Question
                <textarea
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  placeholder="Why is this plan wrong? What should be played instead, and why?"
                />
              </label>
              <label>
                Answer
                <textarea
                  value={answer}
                  onChange={(e) => setAnswer(e.target.value)}
                  placeholder="Your explanation, shown on reveal."
                />
              </label>
            </>
          )}
        </fieldset>

        <fieldset>
          <legend>Organize</legend>
          <label>
            Tags
            <TagInput value={tags} onChange={setTags} suggestions={allTags} />
          </label>
          <label>
            Sets
            <div>
              {sets.map((s) => (
                <label key={s.id} className="checkbox-row">
                  <input type="checkbox" checked={setIDs.includes(s.id)} onChange={() => toggleSet(s.id)} />
                  {s.name}
                </label>
              ))}
              <div className="fen-row" style={{ marginTop: 6 }}>
                <input
                  placeholder="New set name"
                  value={newSetName}
                  onChange={(e) => setNewSetName(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), createSet())}
                />
                <button type="button" className="btn btn--sm" onClick={createSet}>
                  Create set
                </button>
              </div>
            </div>
          </label>
          <label>
            Source note (game link, date, opponent — for your reference)
            <input value={sourceNote} onChange={(e) => setSourceNote(e.target.value)} />
          </label>
        </fieldset>

        {error && <p className="form-error">{error}</p>}

        <div className="editor__actions">
          <button type="button" className="btn btn--primary" onClick={save} disabled={busy || !fenValid}>
            {cardID === null ? 'Create card' : 'Save changes'}
          </button>
          <button type="button" className="btn btn--ghost" onClick={() => navigate('/')}>
            Cancel
          </button>
          {cardID !== null && (
            <button type="button" className="btn btn--ghost btn--danger" onClick={deleteCard}>
              Delete card
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
