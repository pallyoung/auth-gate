declare global {
  var __AUTH_GATE_TEST_NAVIGATE__: ((url: string) => void) | undefined
}

export function navigateTo(url: string) {
  if (typeof window === 'undefined') {
    return
  }

  const testNavigate = globalThis.__AUTH_GATE_TEST_NAVIGATE__
  if (typeof testNavigate === 'function') {
    testNavigate(url)
    return
  }

  window.location.assign(url)
}
