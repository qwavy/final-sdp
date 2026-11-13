import { createContext, useCallback, useContext, useEffect, useState, type DependencyList, type ReactNode } from 'react'
import { toast } from 'sonner'
import { api, errorMessage, type Options } from './api'

// Общее состояние приложения: справочники для форм и счётчик version.
// После любого действия вызываем refresh() — version растёт,
// и все таблицы (useLoad) заново загружают данные с сервера.

type AppState = { version: number; refresh: () => void; options: Options | null }

const AppContext = createContext<AppState>({ version: 0, refresh: () => {}, options: null })

export function AppProvider({ children }: { children: ReactNode }) {
  const [version, setVersion] = useState(0)
  const [options, setOptions] = useState<Options | null>(null)
  const refresh = useCallback(() => setVersion((v) => v + 1), [])

  useEffect(() => {
    api.options().then(setOptions).catch((e) => toast.error(errorMessage(e)))
  }, [])

  return <AppContext.Provider value={{ version, refresh, options }}>{children}</AppContext.Provider>
}

export const useApp = () => useContext(AppContext)

// useLoad загружает данные и перезагружает их после каждого refresh().
export function useLoad<T>(load: () => Promise<T>, deps: DependencyList) {
  const { version } = useApp()
  const [state, setState] = useState<{ data?: T; error?: string; loading: boolean }>({ loading: true })

  useEffect(() => {
    let cancelled = false
    load()
      .then((data) => !cancelled && setState({ data, loading: false }))
      .catch((e) => !cancelled && setState((s) => ({ ...s, error: errorMessage(e), loading: false })))
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [version, ...deps])

  return state
}
