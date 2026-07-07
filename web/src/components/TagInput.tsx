// Free-form tag entry with autocomplete against previously-used tags
// (SPEC.md §6): type to filter, Enter/comma to add, click a suggestion.
import { useMemo, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'

interface Props {
  value: string[]
  onChange: (tags: string[]) => void
  suggestions: string[]
}

export default function TagInput({ value, onChange, suggestions }: Props) {
  const [text, setText] = useState('')
  const [open, setOpen] = useState(false)
  const [focusIdx, setFocusIdx] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const matches = useMemo(() => {
    const q = text.trim().toLowerCase()
    const lower = value.map((t) => t.toLowerCase())
    return suggestions
      .filter((s) => !lower.includes(s.toLowerCase()))
      .filter((s) => q === '' || s.toLowerCase().includes(q))
      .slice(0, 8)
  }, [text, value, suggestions])

  const add = (raw: string) => {
    const tag = raw.trim().replace(/\s+/g, ' ')
    if (!tag) return
    if (value.some((t) => t.toLowerCase() === tag.toLowerCase())) {
      setText('')
      return
    }
    onChange([...value, tag])
    setText('')
    setFocusIdx(0)
  }

  const remove = (tag: string) => onChange(value.filter((t) => t !== tag))

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      if (open && matches.length > 0 && text.trim() === '') add(matches[focusIdx])
      else add(text)
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      setFocusIdx((i) => Math.min(i + 1, matches.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setFocusIdx((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Tab' && open && matches.length > 0 && text.trim() !== '') {
      e.preventDefault()
      add(matches[focusIdx] ?? text)
    } else if (e.key === 'Backspace' && text === '' && value.length > 0) {
      remove(value[value.length - 1])
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  return (
    <div className="tag-input" onClick={() => inputRef.current?.focus()}>
      {value.map((tag) => (
        <span key={tag} className="tag-input__chip">
          {tag}
          <button type="button" aria-label={`Remove tag ${tag}`} onClick={() => remove(tag)}>
            ×
          </button>
        </span>
      ))}
      <input
        ref={inputRef}
        value={text}
        placeholder={value.length === 0 ? 'Add tags…' : ''}
        onChange={(e) => {
          setText(e.target.value)
          setFocusIdx(0)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          // Delay so suggestion clicks land before the list closes.
          setTimeout(() => setOpen(false), 150)
          if (text.trim()) add(text)
        }}
        onKeyDown={onKeyDown}
      />
      {open && matches.length > 0 && (
        <div className="tag-input__suggest">
          {matches.map((s, i) => (
            <button
              key={s}
              type="button"
              className={i === focusIdx ? 'focused' : ''}
              onMouseDown={(e) => {
                e.preventDefault()
                add(s)
              }}
            >
              {s}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
