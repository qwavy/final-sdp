import { useState } from 'react'
import { PlusIcon, SearchIcon } from 'lucide-react'
import { EmptyState, OptionSelect, StatusBadge } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api } from '@/lib/api'
import { dateTime, money } from '@/lib/format'
import { useApp, useLoad } from '@/lib/app'
import { OrderDetailsDialog } from './OrderDetails'
import { OrderFormDialog } from './OrderForm'

export function OrdersPage() {
  const { options } = useApp()
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [creating, setCreating] = useState(false)
  const [selected, setSelected] = useState<string | null>(null)
  const { data: orders, error, loading } = useLoad(() => api.orders(status, search), [status, search])

  const statusOptions = [{ code: 'all', title: 'Все статусы' }, ...(options?.statuses ?? [])]

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-56 flex-1">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Поиск: номер, клиент, телефон, содержимое"
            className="pl-8"
          />
        </div>
        <OptionSelect
          value={status || 'all'}
          onChange={(v) => setStatus(v === 'all' ? '' : v)}
          options={statusOptions}
          className="w-52"
        />
        <Button onClick={() => setCreating(true)}>
          <PlusIcon /> Новый заказ
        </Button>
      </div>

      <Card className="py-0">
        <CardContent className="px-0">
          {error ? (
            <EmptyState>{error}</EmptyState>
          ) : !loading && orders?.length === 0 ? (
            <EmptyState>Заказов не найдено</EmptyState>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-4">Заказ</TableHead>
                  <TableHead>Клиент</TableHead>
                  <TableHead>Доставка</TableHead>
                  <TableHead>Статус</TableHead>
                  <TableHead>Курьер</TableHead>
                  <TableHead className="pr-4 text-right">Цена</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {orders?.map((o) => (
                  <TableRow key={o.id} className="cursor-pointer" onClick={() => setSelected(o.id)}>
                    <TableCell className="pl-4">
                      <div className="font-mono text-xs">{o.id}</div>
                      <div className="max-w-48 truncate text-muted-foreground">{o.description}</div>
                    </TableCell>
                    <TableCell>{o.customerName}</TableCell>
                    <TableCell>
                      {o.deliveryTitle}
                      <div className="text-xs text-muted-foreground">{o.districtName}</div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={o.status} title={o.statusTitle} />
                      <div className="mt-1 text-xs text-muted-foreground">{dateTime(o.updatedAt)}</div>
                    </TableCell>
                    <TableCell>{o.courierName ?? <span className="text-muted-foreground">—</span>}</TableCell>
                    <TableCell className="pr-4 text-right font-medium tabular-nums">{money(o.price)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <OrderFormDialog open={creating} onOpenChange={setCreating} onSaved={(o) => setSelected(o.id)} />
      <OrderDetailsDialog orderId={selected} onClose={() => setSelected(null)} />
    </div>
  )
}
