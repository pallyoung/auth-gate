import React, { useEffect, useRef } from 'react'
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
  sm: 'md:max-w-[400px]',
  md: 'md:max-w-[560px]',
  lg: 'md:max-w-[720px]',
}

export function Modal({
  open,
  onClose,
  modalSize = 'md',
  title,
  children,
}: ModalProps) {
  const previousActiveElement = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (open) {
      previousActiveElement.current = document.activeElement as HTMLElement
      document.body.style.overflow = 'hidden'
      
      // Focus trap - focus the modal
      const modal = document.querySelector('[role="dialog"]') as HTMLElement
      if (modal) {
        modal.focus()
      }
    } else {
      document.body.style.overflow = ''
      
      // Restore focus
      if (previousActiveElement.current) {
        previousActiveElement.current.focus()
      }
    }

    return () => {
      document.body.style.overflow = ''
    }
  }, [open])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onClose()
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
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
      <div className="absolute inset-0 bg-black/50" aria-hidden="true" />
      <div className="flex items-end md:items-center justify-center min-h-screen p-0 md:p-4">
        <div
          className={cn(
            'relative w-full bg-[var(--bg-elevated)]',
            'rounded-t-[var(--radius-xl)] md:rounded-[var(--radius-xl)]',
            'shadow-[var(--shadow-xl)] animate-modal-enter',
            'max-h-[90vh] overflow-y-auto',
            'focus:outline-none',
            sizeStyles[modalSize]
          )}
          onClick={(e) => e.stopPropagation()}
          tabIndex={-1}
        >
          {title && (
            <div className="px-6 py-4 border-b border-[var(--border-default)] sticky top-0 bg-[var(--bg-elevated)] z-10">
              <h2 id="modal-title" className="text-[var(--text-xl)] font-semibold text-[var(--text-primary)]">
                {title}
              </h2>
            </div>
          )}
          <div className="p-4 md:p-6">{children}</div>
        </div>
      </div>
    </div>
  )
}

export function ModalFooter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col-reverse md:flex-row justify-end gap-2 mt-6 pt-4 border-t border-[var(--border-default)]">
      {children}
    </div>
  )
}
