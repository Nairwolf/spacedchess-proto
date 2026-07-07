import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { Tag } from '../api'

export default function Tags() {
  const [tags, setTags] = useState<Tag[]>([])
  const [editing, setEditing] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [error, setError] = useState('')

  const load = () => api.listTags().then(setTags).catch((e) => setError(e.message))
  useEffect(() => {
    load()
  }, [])

  const rename = async (id: number) => {
    setError('')
    try {
      await api.renameTag(id, editName)
      setEditing(null)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not rename tag')
    }
  }

  const remove = async (t: Tag) => {
    if (!window.confirm(`Delete tag "${t.name}"? It is removed from ${t.card_count} card${t.card_count === 1 ? '' : 's'}.`))
      return
    setError('')
    try {
      await api.deleteTag(t.id)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not delete tag')
    }
  }

  return (
    <div className="manage">
      <h1>Tags</h1>
      <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>
        Tags are created while authoring cards. Rename or delete them here.
      </p>
      {error && <p className="form-error">{error}</p>}
      {tags.length === 0 ? (
        <p className="empty-state">No tags yet. Add tags while creating cards.</p>
      ) : (
        tags.map((t) => (
          <div key={t.id} className="manage__row">
            {editing === t.id ? (
              <>
                <input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && rename(t.id)}
                  autoFocus
                />
                <button type="button" className="btn btn--sm" onClick={() => rename(t.id)}>
                  Save
                </button>
                <button type="button" className="btn btn--ghost btn--sm" onClick={() => setEditing(null)}>
                  Cancel
                </button>
              </>
            ) : (
              <>
                <Link className="manage__row-name" to={`/?tag=${encodeURIComponent(t.name)}`}>
                  {t.name}
                </Link>
                <span className="manage__row-count">
                  {t.card_count} card{t.card_count === 1 ? '' : 's'}
                </span>
                <button
                  type="button"
                  className="btn btn--ghost btn--sm"
                  onClick={() => {
                    setEditing(t.id)
                    setEditName(t.name)
                  }}
                >
                  Rename
                </button>
                <button type="button" className="btn btn--ghost btn--sm btn--danger" onClick={() => remove(t)}>
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
