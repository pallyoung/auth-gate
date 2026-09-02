import React, { useEffect, useId, useRef } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../../lib/utils'

type ModalSize = 'sm' | 'md' | 'lg'

interface ModalProps {
  open: boolean
  onClose: () => void
  modalSize?: ModalSize
  title?: string
  children: React.ReactNode
  dialogRole?: 'dialog' | 'alertdialog'
  showCloseButton?: boolean
  closeDisabled?: boolean
  inactive?: boolean
}

const openModalStack: symbol[] = []
let bodyScrollLockCount = 0
let bodyOverflowBeforeLock = ''

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
  dialogRole = 'dialog',
  showCloseButton = true,
  closeDisabled = false,
  inactive = false,
}: ModalProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const modalId = useRef(Symbol('modal'))
  const previousActiveElement = useRef<HTMLElement | null>(null)
  const rootRef = useRef<HTMLDivElement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)
  const getFocusableElements = React.useCallback(() => {
    if (!panelRef.current) {
      return [] as HTMLElement[]
    }

    return Array.from(
      panelRef.current.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      )
    )
  }, [])

  useEffect(() => {
    if (inactive) rootRef.current?.setAttribute('inert', '')
    else rootRef.current?.removeAttribute('inert')
  }, [inactive, open])

  useEffect(() => {
    if (!open) return

    previousActiveElement.current = document.activeElement as HTMLElement
    openModalStack.push(modalId.current)
    if (bodyScrollLockCount === 0) {
      bodyOverflowBeforeLock = document.body.style.overflow
    }
    bodyScrollLockCount += 1
    document.body.style.overflow = 'hidden'
    const [firstFocusableElement] = getFocusableElements()
    ;(firstFocusableElement ?? panelRef.current)?.focus()

    return () => {
      const stackIndex = openModalStack.lastIndexOf(modalId.current)
      if (stackIndex >= 0) openModalStack.splice(stackIndex, 1)
      bodyScrollLockCount = Math.max(0, bodyScrollLockCount - 1)
      if (bodyScrollLockCount === 0) {
        document.body.style.overflow = bodyOverflowBeforeLock
      }
      previousActiveElement.current?.focus()
    }
  }, [getFocusableElements, open])

  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (!open) {
        return
      }
      if (openModalStack[openModalStack.length - 1] !== modalId.current) {
        return
      }

      if (event.key === 'Escape') {
        event.preventDefault()
        if (!closeDisabled) onClose()
        return
      }

      if (event.key !== 'Tab') {
        return
      }

      const focusableElements = getFocusableElements()
      if (focusableElements.length === 0) {
        return
      }

      const [firstFocusableElement] = focusableElements
      const lastFocusableElement = focusableElements[focusableElements.length - 1]
      const activeElement = document.activeElement as HTMLElement | null

      if (!activeElement || !panelRef.current?.contains(activeElement)) {
        event.preventDefault()
        firstFocusableElement.focus()
        return
      }

      if (event.shiftKey && activeElement === firstFocusableElement) {
        event.preventDefault()
        lastFocusableElement.focus()
        return
      }

      if (!event.shiftKey && activeElement === lastFocusableElement) {
        event.preventDefault()
        firstFocusableElement.focus()
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('keydown', handleEscape)
    }
  }, [closeDisabled, open, onClose])

  if (!open) return null

  return (
    <div
      ref={rootRef}
      role={dialogRole}
      aria-modal="true"
      aria-labelledby={title ? titleId : undefined}
      aria-hidden={inactive || undefined}
      className="fixed inset-0 z-[var(--z-modal-backdrop)]"
      onClick={() => {
        if (!closeDisabled && openModalStack[openModalStack.length - 1] === modalId.current) onClose()
      }}
    >
      <div className="absolute inset-0 bg-[rgba(0,0,0,0.6)] backdrop-blur-sm" aria-hidden="true" />
      <div className="flex min-h-screen items-end justify-center p-0 md:items-center md:p-6">
        <div
          ref={panelRef}
          className={cn(
            'relative w-full rounded-t-[16px] border border-[var(--border-default)] bg-[var(--bg-elevated)] shadow-[var(--shadow-xl)] outline-none animate-modal-enter',
            'max-h-[92vh] overflow-y-auto md:rounded-[16px]',
            sizeStyles[modalSize]
          )}
          onClick={(event) => event.stopPropagation()}
          tabIndex={-1}
        >
          {title && (
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-[var(--border-default)] bg-[var(--bg-elevated)] px-5 py-4">
              <h2 id={titleId} className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
                {title}
              </h2>
              {showCloseButton && (
                <button
                  type="button"
                  onClick={onClose}
                  disabled={closeDisabled}
                  className="flex h-9 w-9 items-center justify-center rounded-[8px] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)] disabled:cursor-not-allowed disabled:opacity-50"
                  aria-label={t('actions.closeModal')}
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          )}
          <div className="p-5">{children}</div>
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
