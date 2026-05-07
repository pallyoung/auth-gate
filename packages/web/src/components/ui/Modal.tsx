import React, { useEffect, useRef } from 'react'
import { X } from 'lucide-react'
import { cn } from '../../lib/utils'

type ModalSize = 'sm' | 'md' | 'lg'

interface ModalProps {
  open: boolean
  onClose: () => void
  modalSize?: ModalSize
  title?: string
  children: React.ReactNode
}

const sizeStyles: Record<ModalSize, string> = {
  sm: 'md:max-w-[440px]',
  md: 'md:max-w-[620px]',
  lg: 'md:max-w-[820px]',
}

export function Modal({
  open,
  onClose,
  modalSize = 'md',
  title,
  children,
}: ModalProps) {
  const previousActiveElement = useRef<HTMLElement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (open) {
      previousActiveElement.current = document.activeElement as HTMLElement
      document.body.style.overflow = 'hidden'
      panelRef.current?.focus()
      return
    }

    document.body.style.overflow = ''
    previousActiveElement.current?.focus()
  }, [open])

  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && open) {
        onClose()
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => {
      document.body.style.overflow = ''
      document.removeEventListener('keydown', handleEscape)
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? 'modal-title' : undefined}
      className="fixed inset-0 z-[var(--z-modal-backdrop)]"
      onClick={onClose}
    >
      <div className="absolute inset-0 bg-[rgba(15,23,34,0.52)] backdrop-blur-md" aria-hidden="true" />
      <div className="flex min-h-screen items-end justify-center p-0 md:items-center md:p-6">
        <div
          ref={panelRef}
          className={cn(
            'relative w-full rounded-t-[30px] border border-white/10 bg-[var(--bg-elevated)] shadow-[var(--shadow-xl)] outline-none animate-modal-enter',
            'max-h-[92vh] overflow-y-auto md:rounded-[30px]',
            sizeStyles[modalSize]
          )}
          onClick={(event) => event.stopPropagation()}
          tabIndex={-1}
        >
          {title && (
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-[var(--border-default)] bg-[var(--bg-elevated)] px-6 py-5 backdrop-blur-xl">
              <h2 id="modal-title" className="text-xl font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
                {title}
              </h2>
              <button
                onClick={onClose}
                className="flex h-10 w-10 items-center justify-center rounded-full text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                aria-label="Close modal"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          )}
          <div className="p-5 md:p-6">{children}</div>
        </div>
      </div>
    </div>
  )
}

export function ModalFooter({ children }: { children: React.ReactNode }) {
  return (
    <div className="mt-6 flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
      {children}
    </div>
  )
}
