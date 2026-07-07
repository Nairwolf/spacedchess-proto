import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { CardSet } from '../api'

export default function Sets() {
  const [sets, setSets] = useState<CardSet[]>([])
  const [newName, setNewName] = useState('')
  const [editing, setEditing] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [error, setError] = useState('')

  const load = () => api.listSets().then(setSets).catch((e) => setError(e.message))
  useEffect(() => {
    load()
  }, [])

  const create = async () => {
    if (!newName.trim()) return
    setError('')
    try {
      await api.createSet(newName)
      setNewName('')
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not create set')
    }
  }

  const rename = async (id: number) => {
    setError('')
    try {
      await api.renameSet(id, editName)
      setEditing(null)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not rename set')
    }
  }

  const remove = async (s: CardSet) => {
    if (!window.confirm(`Delete set "${s.name}"? Cards in it are kept.`)) return
    setError('')
    try {
      await api.deleteSet(s.id)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not delete set')
    }
  }

  return (
    <div className="manage">
      <h1>Sets</h1>
      <div className="manage__new">
        <input
          placeholder="New set name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && create()}
        />
        <button type="button" className="btn btn--primary" onClick={create}>
          Create
        </button>
      </div>
      {error && <p className="form-error">{error}</p>}
      {sets.length === 0 ? (
        <p className="empty-state">No sets yet. A set is a saved collection of cards you can review as a unit.</p>
      ) : (
        sets.map((s) => (
          <div key={s.id} className="manage__row">
            {editing === s.id ? (
              <>
                <input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && rename(s.id)}
                  autoFocus
                />
                <button type="button" className="btn btn--sm" onClick={() => rename(s.id)}>
                  Save
                </button>
                <button type="button" className="btn btn--ghost btn--sm" onClick={() => setEditing(null)}>
                  Cancel
                </button>
              </>
            ) : (
              <>
                <Link className="manage__row-name" to={`/?set_id=${s.id}`}>
                  {s.name}
                </Link>
                <span className="manage__row-count">
                  {s.card_count} card{s.card_count === 1 ? '' : 's'}
                </span>
                <Link className="btn btn--sm" to={`/review?set_id=${s.id}`}>
                  Review
                </Link>
                <button
                  type="button"
                  className="btn btn--ghost btn--sm"
                  onClick={() => {
                    setEditing(s.id)
                    setEditName(s.name)
                  }}
                >
                  Rename
                </button>
                <button type="button" className="btn btn--ghost btn--sm btn--danger" onClick={() => remove(s)}>
                  Delete
                </button>
              </>
            )}
          </div>
        ))
      )}
    </div>
  )
}
