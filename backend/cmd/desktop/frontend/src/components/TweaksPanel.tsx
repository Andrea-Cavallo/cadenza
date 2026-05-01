import { ReactNode, useCallback, useEffect, useRef, useState } from 'react'

type TweakValue = string | number | boolean
type TweakMap = Record<string, TweakValue>

interface TweaksPanelProps {
  title: string
  children: ReactNode
}

interface TweakSectionProps {
  label: string
  children: ReactNode
}

interface TweakRadioProps {
  label?: string
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
}

export function useTweaks<T extends TweakMap>(defaults: T): [T, (key: keyof T, value: T[keyof T]) => void] {
  const [values, setValues] = useState(defaults)
  const setTweak = useCallback((key: keyof T, value: T[keyof T]) => {
    setValues(prev => ({ ...prev, [key]: value }))
  }, [])
  return [values, setTweak]
}

export function TweaksPanel({ title, children }: TweaksPanelProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'e') {
        setOpen(prev => !prev)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  if (!open) return null

  return (
    <div ref={ref} className="twk-panel" style={{ right: 16, bottom: 16 }}>
      <div className="twk-hd">
        <b>{title}</b>
        <button className="twk-x" type="button" aria-label="Close tweaks" onClick={() => setOpen(false)}>x</button>
      </div>
      <div className="twk-body">{children}</div>
    </div>
  )
}

export function TweakSection({ label, children }: TweakSectionProps) {
  return (
    <>
      <div className="twk-sect">{label}</div>
      {children}
    </>
  )
}

export function TweakRadio({ label, value, onChange, options }: TweakRadioProps) {
  return (
    <div className="twk-row">
      {label && <div className="twk-lbl"><span>{label}</span></div>}
      <div className="twk-seg">
        {options.map(option => (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={option.value === value}
            style={{ background: option.value === value ? 'rgba(255,255,255,.9)' : 'transparent' }}
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  )
}
