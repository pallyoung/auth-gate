import React, { forwardRef, useCallback, useEffect, useId, useRef, useState } from 'react'
import { Check, ChevronDown } from 'lucide-react'
import { cn } from '../../lib/utils'

type SelectSize = 'sm' | 'md' | 'lg'

interface SelectOption {
  value: string
  label: string
}

interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, 'onChange'> {
  label?: string
  error?: string
  hint?: string
  selectSize?: SelectSize
  options: SelectOption[]
  onChange?: (event: { target: { value: string } }) => void
}

const sizeStyles: Record<SelectSize, { trigger: string; dropdown: string }> = {
  sm: { trigger: 'h-10 text-sm', dropdown: 'text-sm' },
  md: { trigger: 'h-12 text-sm', dropdown: 'text-sm' },
  lg: { trigger: 'h-14 text-base', dropdown: 'text-base' },
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      label,
      error,
      hint,
      selectSize = 'md',
      options,
      className,
      id: idProp,
      disabled,
      value,
      onChange,
      name,
      ...rest
    },
    _ref
  ) => {
    const autoId = useId()
    const selectId = idProp || label?.toLowerCase().replace(/\s+/g, '-') || autoId
    const listId = `${selectId}-listbox`
    const errorId = error ? `${selectId}-error` : undefined
    const hintId = hint && !error ? `${selectId}-hint` : undefined
    const size = sizeStyles[selectSize]

    const [open, setOpen] = useState(false)
    const [activeIndex, setActiveIndex] = useState(-1)
    const rootRef = useRef<HTMLDivElement>(null)
    const triggerRef = useRef<HTMLButtonElement>(null)
    const listRef = useRef<HTMLUListElement>(null)

    const selectedOption = options.find((o) => o.value === value) ?? options[0]

    const commit = useCallback(
      (val: string) => {
        onChange?.({ target: { value: val } })
        setOpen(false)
        triggerRef.current?.focus()
      },
      [onChange]
    )

    // Click outside → close
    useEffect(() => {
      if (!open) return
      const handler = (e: MouseEvent) => {
        if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
          setOpen(false)
        }
      }
      document.addEventListener('mousedown', handler)
      return () => document.removeEventListener('mousedown', handler)
    }, [open])

    // Scroll active item into view
    useEffect(() => {
      if (!open || activeIndex < 0) return
      const el = listRef.current?.children[activeIndex] as HTMLElement | undefined
      el?.scrollIntoView({ block: 'nearest' })
    }, [activeIndex, open])

    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (disabled) return

      switch (e.key) {
        case 'Enter':
        case ' ':
          e.preventDefault()
          if (open && activeIndex >= 0) {
            commit(options[activeIndex].value)
          } else {
            setOpen(true)
            setActiveIndex(options.findIndex((o) => o.value === value))
          }
          break
        case 'ArrowDown':
          e.preventDefault()
          if (!open) {
            setOpen(true)
            setActiveIndex(0)
          } else {
            setActiveIndex((i) => Math.min(i + 1, options.length - 1))
          }
          break
        case 'ArrowUp':
          e.preventDefault()
          if (!open) {
            setOpen(true)
            setActiveIndex(options.length - 1)
          } else {
            setActiveIndex((i) => Math.max(i - 1, 0))
          }
          break
        case 'Escape':
          setOpen(false)
          triggerRef.current?.focus()
          break
        case 'Tab':
          setOpen(false)
          break
      }
    }

    return (
      <div className="flex flex-col gap-2" ref={rootRef}>
        {label && (
          <label
            htmlFor={selectId}
            className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]"
          >
            {label}
          </label>
        )}

        <div className={cn('relative', size.trigger)}>
          {/* Hidden native select for form submission / accessibility */}
          <select
            ref={_ref}
            name={name}
            tabIndex={-1}
            aria-hidden="true"
            className="pointer-events-none absolute h-0 w-0 opacity-0"
            value={value}
            onChange={() => {}}
            {...rest}
          >
            {options.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>

          {/* Trigger button */}
          <button
            ref={triggerRef}
            id={selectId}
            type="button"
            role="combobox"
            aria-haspopup="listbox"
            aria-expanded={open}
            aria-controls={listId}
            aria-invalid={!!error}
            aria-describedby={errorId || hintId}
            disabled={disabled}
            onClick={() => {
              if (!disabled) {
                setOpen((o) => !o)
                if (!open) setActiveIndex(options.findIndex((o) => o.value === value))
              }
            }}
            onKeyDown={handleKeyDown}
            className={cn(
              'flex h-full w-full items-center gap-2 rounded-[12px] border pl-4',
              'bg-[var(--bg-input)] text-left text-[var(--text-primary)]',
              'transition-all duration-[var(--duration-normal)]',
              // border
              'border-[var(--border-default)]',
              'hover:border-[var(--border-strong)]',
              // focus
              'focus:border-[var(--primary-500)] focus:outline-none focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]',
              // open
              open && 'border-[var(--primary-500)] ring-2 ring-[rgba(15,143,139,0.2)]',
              // disabled
              'disabled:cursor-not-allowed disabled:opacity-50',
              // error
              error && 'border-[var(--error)] focus:border-[var(--error)] focus:ring-[var(--error-light)]',
              size.trigger,
              className
            )}
          >
            <span className="flex-1 truncate">
              {selectedOption ? selectedOption.label : <span className="text-[var(--text-subtle)]">—</span>}
            </span>
            <ChevronDown
              className={cn(
                'h-4 w-4 shrink-0 text-[var(--text-subtle)] transition-transform duration-[var(--duration-normal)]',
                open && 'rotate-180 text-[var(--primary-500)]'
              )}
            />
          </button>

          {/* Dropdown list */}
          {open && (
            <ul
              ref={listRef}
              id={listId}
              role="listbox"
              aria-labelledby={selectId}
              className={cn(
                'absolute left-0 top-full z-[var(--z-dropdown)] mt-1.5 w-full',
                'overflow-hidden rounded-[14px] border border-[var(--border-strong)]',
                'bg-[var(--bg-elevated)] shadow-[var(--shadow-lg)]',
                'backdrop-blur-xl',
                'animate-[fadeSlideDown_var(--duration-normal)_var(--ease-out)]',
                size.dropdown
              )}
              style={{ maxHeight: '16rem', overflowY: 'auto' }}
            >
              {options.map((option, i) => {
                const isSelected = option.value === value
                const isActive = i === activeIndex
                return (
                  <li
                    key={option.value}
                    role="option"
                    aria-selected={isSelected}
                    onMouseEnter={() => setActiveIndex(i)}
                    onClick={() => commit(option.value)}
                    className={cn(
                      'flex cursor-pointer items-center gap-3 px-4 py-2.5',
                      'transition-colors duration-[var(--duration-fast)]',
                      // base text
                      'text-[var(--text-secondary)]',
                      // hover / active
                      isActive && 'bg-[var(--bg-hover)] text-[var(--text-primary)]',
                      // selected
                      isSelected && !isActive && 'text-[var(--primary-500)]',
                      isSelected && isActive && 'text-[var(--primary-500)]',
                      // first / last rounding
                      i === 0 && 'pt-3',
                      i === options.length - 1 && 'pb-3'
                    )}
                  >
                    <span className="flex-1 truncate">{option.label}</span>
                    {isSelected && (
                      <Check className="h-4 w-4 shrink-0 text-[var(--primary-500)]" />
                    )}
                  </li>
                )
              })}
            </ul>
          )}
        </div>

        {error && (
          <span id={errorId} role="alert" className="text-xs font-medium text-[var(--error)]">
            {error}
          </span>
        )}
        {hint && !error && (
          <span id={hintId} className="text-xs text-[var(--text-muted)]">
            {hint}
          </span>
        )}
      </div>
    )
  }
)

Select.displayName = 'Select'
