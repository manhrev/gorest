import { useState } from 'react'
import ClientCredentialsPage from './pages/ClientCredentialsPage'
import AuthCodePage from './pages/AuthCodePage'
import AuthCodePublicPage from './pages/AuthCodePublicPage'

const PAGES = {
  'client-credentials': { label: 'Client Credentials', Component: ClientCredentialsPage },
  'auth-code': { label: 'Authorization Code (Confidential)', Component: AuthCodePage },
  'auth-code-public': { label: 'Authorization Code (Public)', Component: AuthCodePublicPage },
} as const

type PageId = keyof typeof PAGES

export default function App() {
  const [page, setPage] = useState<PageId>('client-credentials')
  const { Component } = PAGES[page]

  return (
    <>
      <nav className="page-nav">
        {(Object.keys(PAGES) as PageId[]).map((id) => (
          <button
            key={id}
            className={`page-nav-btn${id === page ? ' selected' : ''}`}
            onClick={() => setPage(id)}
          >
            {PAGES[id].label}
          </button>
        ))}
      </nav>
      <Component />
    </>
  )
}
