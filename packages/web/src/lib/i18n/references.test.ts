import { describe, expect, it } from 'vitest'
import ts from 'typescript'
import { createAppI18n } from './index'
import { resources } from './resources'

// Check the keys the UI actually requests, including keys missing in BOTH locales.
// Resource parity alone cannot catch that regression.
const sources = import.meta.glob('../../{pages,components}/**/*.tsx', {
  query: '?raw', import: 'default', eager: true,
}) as Record<string, string>

describe('UI translation references', () => {
  it('keeps all namespace keys aligned, including dynamically selected keys', () => {
    const keys = (value: unknown, prefix = ''): string[] => Object.entries(value as Record<string, unknown>)
      .flatMap(([key, child]) => typeof child === 'object' ? keys(child, `${prefix}${key}.`) : [`${prefix}${key}`])
      .sort()
    for (const namespace of Object.keys(resources.en)) {
      expect(keys(resources['zh-CN'][namespace]), namespace).toEqual(keys(resources.en[namespace]))
    }
  })

  it.each(['en', 'zh-CN'] as const)('resolves literal UI keys without a fallback in %s', async (locale) => {
    const i18n = await createAppI18n(locale)
    const missing: string[] = []
    for (const [path, source] of Object.entries(sources)) {
      if (path.endsWith('.test.tsx')) continue
      const file = ts.createSourceFile(path, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
      let namespace = 'common'
      const requests: { key: string; line: number }[] = []
      const visit = (node: ts.Node) => {
        if (ts.isCallExpression(node) && ts.isIdentifier(node.expression)) {
          const [arg] = node.arguments
          if (node.expression.text === 'useTranslation' && arg) {
            const first = ts.isArrayLiteralExpression(arg) ? arg.elements[0] : arg
            if (first && ts.isStringLiteral(first)) namespace = first.text
          }
          if (node.expression.text === 't' && arg && ts.isStringLiteral(arg)) {
            requests.push({ key: arg.text, line: file.getLineAndCharacterOfPosition(node.getStart()).line + 1 })
          }
        }
        ts.forEachChild(node, visit)
      }
      visit(file)
      for (const { key, line } of requests) {
        if (!i18n.exists(key, { ns: namespace, lng: locale, fallbackLng: false, count: 2 })) {
          missing.push(`${path}:${line} ${namespace}:${key}`)
        }
      }
    }
    expect(Object.keys(sources).length).toBeGreaterThan(20)
    expect(missing).toEqual([])
  })
})
