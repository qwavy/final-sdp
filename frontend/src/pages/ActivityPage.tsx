import { MessageSquareIcon, PackageIcon, RefreshCwIcon, TruckIcon } from 'lucide-react'
import { EmptyState } from '@/components/common'
import { Card, CardContent } from '@/components/ui/card'
import { api } from '@/lib/api'
import { dateTime } from '@/lib/format'
import { useLoad } from '@/lib/live'

// Журнал заполняют наблюдатели (Observer) на сервере:
// ActivityRecorder пишет каждое событие, CustomerNotifier — отправленные SMS.
export function ActivityPage() {
  const { data: entries } = useLoad(() => api.activity(), [])

  return (
    <Card className="py-0">
      <CardContent className="px-0">
        {entries?.length === 0 ? (
          <EmptyState>Событий пока нет</EmptyState>
        ) : (
          <ul className="divide-y">
            {entries?.map((e) => (
              <li key={e.id} className="flex items-start gap-3 px-4 py-2.5 text-sm">
                <span className="mt-0.5 text-muted-foreground">{icon(e.kind)}</span>
                <span className="flex-1">{e.message}</span>
                <span className="shrink-0 tabular-nums text-xs text-muted-foreground">{dateTime(e.at)}</span>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

function icon(kind: string) {
  if (kind.startsWith('sms.')) return <MessageSquareIcon className="size-4" />
  if (kind === 'order.courier_assigned') return <TruckIcon className="size-4" />
  if (kind === 'order.status_changed') return <RefreshCwIcon className="size-4" />
  return <PackageIcon className="size-4" />
}
