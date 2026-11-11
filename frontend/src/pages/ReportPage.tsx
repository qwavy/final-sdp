import { useState } from 'react'
import { DownloadIcon } from 'lucide-react'
import { EmptyState, LoadBar } from '@/components/common'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api } from '@/lib/api'
import { money, time, today } from '@/lib/format'
import { useLoad } from '@/lib/live'

export function ReportPage() {
  const [date, setDate] = useState(today())
  const { data: report, error } = useLoad(() => api.dailyReport(date), [date])

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="w-44" />
        <div className="flex-1" />
        <a href={api.exportUrl(date, 'csv')} download className={buttonVariants({ variant: 'outline' })}>
          <DownloadIcon /> Выгрузить CSV
        </a>
        <a href={api.exportUrl(date, 'json')} download className={buttonVariants({ variant: 'outline' })}>
          <DownloadIcon /> Выгрузить JSON
        </a>
      </div>

      {error && <EmptyState>{error}</EmptyState>}
      {report && (
        <>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
            <Stat label="Всего заказов" value={report.totalOrders} />
            <Stat label="В работе" value={report.activeOrders} />
            <Stat label="Выдано" value={report.deliveredOrders} />
            <Stat label="Выручка" value={money(report.revenue)} />
            <Stat label="Средний чек" value={money(report.averagePrice)} />
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>По способу доставки</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-2 text-sm">
                {report.byDelivery?.length ? (
                  report.byDelivery.map((d) => (
                    <div key={d.type} className="flex justify-between">
                      <span>
                        {d.title} <span className="text-muted-foreground">× {d.count}</span>
                      </span>
                      <span className="tabular-nums">{money(d.revenue)}</span>
                    </div>
                  ))
                ) : (
                  <span className="text-muted-foreground">Нет данных</span>
                )}
                <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 border-t pt-3 text-xs text-muted-foreground">
                  {report.byStatus.map((s) => (
                    <span key={s.status}>
                      {s.title}: <b className="text-foreground">{s.count}</b>
                    </span>
                  ))}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>Курьеры</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-2 text-sm">
                {report.couriers?.map((c) => (
                  <div key={c.id} className="flex items-center justify-between gap-3">
                    <span>
                      {c.name} <span className="text-muted-foreground">· выдано {c.delivered}</span>
                    </span>
                    <LoadBar value={c.activeOrders} max={c.capacity} />
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>

          <Card className="py-0">
            <CardContent className="px-0">
              {!report.orders?.length ? (
                <EmptyState>За этот день заказов нет</EmptyState>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="pl-4">Время</TableHead>
                      <TableHead>Заказ</TableHead>
                      <TableHead>Клиент</TableHead>
                      <TableHead>Доставка</TableHead>
                      <TableHead>Статус</TableHead>
                      <TableHead>Курьер</TableHead>
                      <TableHead className="pr-4 text-right">Цена</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {report.orders.map((o) => (
                      <TableRow key={o.id}>
                        <TableCell className="pl-4 tabular-nums">{time(o.created)}</TableCell>
                        <TableCell className="font-mono text-xs">{o.id}</TableCell>
                        <TableCell>{o.customer}</TableCell>
                        <TableCell>{o.delivery}</TableCell>
                        <TableCell>{o.status}</TableCell>
                        <TableCell>{o.courier || '—'}</TableCell>
                        <TableCell className="pr-4 text-right tabular-nums">{money(o.price)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <Card size="sm">
      <CardContent>
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="mt-1 text-2xl font-semibold tabular-nums">{value}</div>
      </CardContent>
    </Card>
  )
}
