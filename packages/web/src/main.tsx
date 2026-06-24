import React from 'react'
import ReactDOM from 'react-dom/client'
import { I18nextProvider } from 'react-i18next'
import App from './App'
import { i18nPromise } from './lib/i18n'
import './index.css'

if (!window.location.hash) {
  window.location.hash = '/'
}

class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { error: Error | null }
> {
  state = { error: null as Error | null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 32, fontFamily: 'monospace', color: '#d0474b', background: '#fff5f5', minHeight: '100vh' }}>
          <h2 style={{ marginTop: 0 }}>Something went wrong</h2>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.6 }}>
            {this.state.error.message}
            {'\n\n'}
            {this.state.error.stack}
          </pre>
          <button
            onClick={() => { window.location.hash = '/'; window.location.reload() }}
            style={{ marginTop: 16, padding: '8px 16px', cursor: 'pointer' }}
          >
            Go to home
          </button>
        </div>
      )
    }
    return this.props.children
  }
}

async function bootstrap() {
  const i18n = await i18nPromise

  ReactDOM.createRoot(document.getElementById('root')!).render(
    <ErrorBoundary>
      <I18nextProvider i18n={i18n}>
        <App />
      </I18nextProvider>
    </ErrorBoundary>
  )
}

void bootstrap()
