const HTTP_METHOD_TOKEN = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/

export function normalizeCORSMethods(value: string): string | null {
  const methods = value
    .split(',')
    .map((method) => method.trim())
    .filter(Boolean)

  if (methods.some((method) => !HTTP_METHOD_TOKEN.test(method))) {
    return null
  }

  return methods.map((method) => method.toUpperCase()).join(',')
}
