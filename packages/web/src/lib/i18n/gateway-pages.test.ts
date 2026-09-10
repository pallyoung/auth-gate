// @vitest-environment node
import { readFileSync, existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { JSDOM } from 'jsdom'
import { describe, expect, it, vi } from 'vitest'

// Execute the scripts from the actual Go templates, not a copied implementation.
function template(path: string) {
  const source = readFileSync(resolve('../server/internal', path), 'utf8')
  const sharedPath = resolve('../server/internal/pagei18n/runtime.go')
  const shared = existsSync(sharedPath) ? readFileSync(sharedPath, 'utf8') : ''
  return [...source.slice(source.indexOf('const ')).matchAll(/`([^`]*)`|pagei18n\.(Runtime|Switcher)/g)]
    .map(match => match[1] ?? shared.match(new RegExp('const ' + match[2] + ' = `([^`]*)`'))?.[1] ?? '').join('')
}

const pages = [
  ['http/proxy/loginpage.go', 'Gateway Login', '网关登录'],
  ['proxy/notfoundpage.go', 'Route not found', '路由未找到'],
  ['proxy/badgatewaypage.go', 'Service unavailable', '服务暂不可用'],
] as const

function openPage(path: string, options: { stored?: string; browser?: string[]; blocked?: boolean; fetch?: unknown } = {}) {
  return new JSDOM(template(path), {
    url: 'https://gateway.example.test/_authgate/access-login?route_name=Example',
    runScripts: 'dangerously',
    beforeParse(window: any) {
      Object.defineProperty(window.navigator, 'languages', { value: options.browser ?? ['en-US'] })
      if (options.stored) window.localStorage.setItem('auth-gate.locale', options.stored)
      if (options.blocked) Object.defineProperty(window, 'localStorage', { get() { throw new Error('blocked') } })
      window.fetch = options.fetch ?? vi.fn()
    },
  })
}

describe('standalone gateway pages', () => {
  it.each(pages)('%s follows browser locale and persists a manual switch on its own origin', (path, english, chinese) => {
    const dom = openPage(path)
    const doc = dom.window.document
    expect(doc.documentElement.lang).toBe('en')
    expect(doc.querySelector('h1')!.textContent).toContain(english)
    doc.querySelector<HTMLButtonElement>('[data-locale="zh-CN"]')!.click()
    expect(doc.documentElement.lang).toBe('zh-CN')
    expect(doc.querySelector('h1')!.textContent).toContain(chinese)
    expect(dom.window.localStorage.getItem('auth-gate.locale')).toBe('zh-CN')
    expect(doc.title).toContain(chinese)
    dom.window.close()
  })

  it.each(pages)('%s prefers a stored choice and supports unavailable storage', (path, english, chinese) => {
    const stored = openPage(path, { stored: 'en', browser: ['zh-CN'] })
    expect(stored.window.document.querySelector('h1')!.textContent).toContain(english)
    stored.window.close()
    const blocked = openPage(path, { blocked: true, browser: ['fr-FR', 'zh-TW'] })
    expect(blocked.window.document.querySelector('h1')!.textContent).toContain(chinese)
    blocked.window.document.querySelector<HTMLButtonElement>('[data-locale="en"]')!.click()
    expect(blocked.window.document.documentElement.lang).toBe('en')
    blocked.window.close()
  })

  it('localizes login validation and failures without displaying raw backend messages', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: false, json: async () => ({ error: { code: 'unexpected', message: 'RAW_BACKEND_ERROR' } }) })
    const dom = openPage('http/proxy/loginpage.go', { stored: 'en', fetch })
    const doc = dom.window.document
    const submit = () => doc.getElementById('login-form')!.dispatchEvent(new dom.window.Event('submit', { cancelable: true }))
    submit()
    expect(fetch).not.toHaveBeenCalled()
    expect(doc.getElementById('error')!.textContent).toContain('Enter your username and password')
    ;(doc.getElementById('username') as HTMLInputElement).value = 'test'
    ;(doc.getElementById('password') as HTMLInputElement).value = 'test-password'
    submit()
    await vi.waitFor(() => expect(doc.getElementById('error')!.textContent).toContain('Sign-in failed'))
    doc.querySelector<HTMLButtonElement>('[data-locale="zh-CN"]')!.click()
    expect(doc.getElementById('error')!.textContent).toContain('登录失败')
    expect(doc.body.textContent).not.toContain('RAW_BACKEND_ERROR')
    expect(doc.getElementById('route-info')!.textContent).toContain('Example')
    dom.window.close()
  })
})
