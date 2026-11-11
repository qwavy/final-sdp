import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from '@/components/ui/sonner'
import { LiveProvider } from '@/lib/live'
import App from './App.tsx'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LiveProvider>
      <App />
      <Toaster position="bottom-right" />
    </LiveProvider>
  </StrictMode>,
)
