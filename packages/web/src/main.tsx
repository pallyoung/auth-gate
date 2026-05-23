import React from 'react'
import ReactDOM from 'react-dom/client'
import { I18nextProvider } from 'react-i18next'
import App from './App'
import { i18nPromise } from './lib/i18n'
import './index.css'

if (!window.location.hash) {
  window.location.hash = '/'
}

async function bootstrap() {
  const i18n = await i18nPromise

  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <I18nextProvider i18n={i18n}>
        <App />
      </I18nextProvider>
    </React.StrictMode>
  )
}

void bootstrap()
