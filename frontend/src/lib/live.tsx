import { createContext, useContext, useEffect, useState, type DependencyList, type ReactNode } from 'react'
import { toast } from 'sonner'
import { api, errorMessage, type Options } from './api'

// Живые обновления: браузер подписан на /api/events (Server-Sent Events).
// Когда на сервере происходит событие (Observer), здесь показывается
// уведомление и увеличивается version — все таблицы перезагружают данные.

type LiveState = { version: number; connected: boolean; options: Options | null }

const LiveContext = createContext<LiveState>({ version: 0, connected: false, options: null })

export function LiveProvider({ children }: { children: ReactNode }) {
  const [version, setVersion] = useState(0)
  const [connected, setConnected] = useState(false)
  const [options, setOptions] = useState<Options | null>(null)

  useEffect(() => {
    api.options().then(setOptions).catch((e) => toast.error(errorMessage(e)))
  }, [])

  useEffect(() => {
    const source = new EventSource('/api/events')
    source.onopen = () => setConnected(true)
    source.onerror = () => setConnected(false)
    source.onmessage = (event) => {
      const message = JSON.parse(event.data) as { type: string; message: string }
      toast(message.message, { description: eventLabel(message.type) })
      setVersion((v) => v + 1)
    }
    return () => source.close()
  }, [])

  return <LiveContext.Provider value={{ version, connected, options }}>{children}</LiveContext.Provider>
}

function eventLabel(type: string): string {
  switch (type) {
    case 'order.status_changed':
      return 'Статус изменён · клиенту отправлено SMS'
    case 'order.courier_assigned':
      return 'Курьер назначен'
    case 'order.created':
      return 'Новый заказ'
    case 'order.deleted':
      return 'Заказ удалён'
    default:
      return 'Событие'
  }
}

export const useLive = () => useContext(LiveContext)

// useLoad загружает данные и перезагружает их при каждом событии с сервера.
export function useLoad<T>(load: () => Promise<T>, deps: DependencyList) {
  const { version } = useLive()
  const [tick, setTick] = useState(0)
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
  }, [version, tick, ...deps])

  return { ...state, reload: () => setTick((t) => t + 1) }
}
