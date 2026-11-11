import { useState } from 'react'
import { PackageIcon } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useLive } from '@/lib/live'
import { cn } from '@/lib/utils'
import { ActivityPage } from '@/pages/ActivityPage'
import { CouriersPage } from '@/pages/CouriersPage'
import { CustomersPage } from '@/pages/CustomersPage'
import { OrdersPage } from '@/pages/orders/OrdersPage'
import { ReportPage } from '@/pages/ReportPage'

const tabs = [
  { value: 'orders', label: 'Заказы', page: <OrdersPage /> },
  { value: 'customers', label: 'Клиенты', page: <CustomersPage /> },
  { value: 'couriers', label: 'Курьеры и точки', page: <CouriersPage /> },
  { value: 'activity', label: 'Журнал событий', page: <ActivityPage /> },
  { value: 'report', label: 'Отчёт за день', page: <ReportPage /> },
]

export default function App() {
  const { connected } = useLive()
  const [tab, setTab] = useState(() => {
    try {
      return localStorage.getItem('tab') ?? 'orders'
    } catch {
      return 'orders'
    }
  })

  const changeTab = (value: string) => {
    setTab(value)
    try {
      localStorage.setItem('tab', value)
    } catch {
      // хранилище браузера недоступно — просто не запоминаем вкладку
    }
  }

  return (
    <div className="mx-auto grid max-w-6xl gap-6 px-4 py-6">
      <header className="flex items-center gap-3">
        <div className="flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <PackageIcon className="size-5" />
        </div>
        <div className="flex-1">
          <h1 className="text-lg font-semibold leading-tight">Parcel Hub</h1>
          <p className="text-sm text-muted-foreground">Пункт выдачи и курьерская доставка</p>
        </div>
        <span className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className={cn('size-2 rounded-full', connected ? 'bg-emerald-500' : 'bg-rose-500')} />
          {connected ? 'онлайн' : 'нет связи с сервером'}
        </span>
      </header>

      <Tabs value={tab} onValueChange={(v) => changeTab(String(v))}>
        <TabsList className="flex-wrap">
          {tabs.map((t) => (
            <TabsTrigger key={t.value} value={t.value}>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>
        {tabs.map((t) => (
          <TabsContent key={t.value} value={t.value} className="mt-4">
            {t.page}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
