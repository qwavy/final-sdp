import { useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { OptionSelect, StatusBadge } from '@/components/common'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { api, errorMessage, type Order } from '@/lib/api'
import { dateTime, money } from '@/lib/format'
import { useApp, useLoad } from '@/lib/app'
import { OrderFormDialog } from './OrderForm'

export function OrderDetailsDialog({ orderId, onClose }: { orderId: string | null; onClose: () => void }) {
  return (
    <Dialog open={orderId !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-2xl">
        {orderId && <OrderDetails orderId={orderId} onDeleted={onClose} />}
      </DialogContent>
    </Dialog>
  )
}

function OrderDetails({ orderId, onDeleted }: { orderId: string; onDeleted: () => void }) {
  const { options, refresh } = useApp()
  const { data: order, error } = useLoad(() => api.order(orderId), [orderId])
  const [selector, setSelector] = useState('nearest')
  const [busy, setBusy] = useState(false)
  const [editing, setEditing] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  if (error) return <p className="text-sm text-destructive">{error}</p>
  if (!order || !options) return <p className="text-sm text-muted-foreground">Загрузка…</p>

  // run выполняет действие, показывает результат и обновляет все таблицы.
  async function run(action: () => Promise<Order>) {
    setBusy(true)
    try {
      const updated = await action()
      toast.success(`Заказ ${updated.id}: ${updated.statusTitle}`, { description: 'Клиенту отправлено SMS' })
      refresh()
    } catch (e) {
      toast.error(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  const strategyTitle = options.pricingStrategies.find((s) => s.code === order.pricingStrategy)?.title
  const extraTitles = order.extras.map((code) => options.extras.find((e) => e.code === code)?.title ?? code)

  return (
    <>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-3">
          <span className="font-mono">{order.id}</span>
          <StatusBadge status={order.status} title={order.statusTitle} />
        </DialogTitle>
      </DialogHeader>

      <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
        <Info label="Клиент">
          {order.customerName}
          <div className="text-muted-foreground">{order.customerPhone}</div>
        </Info>
        <Info label="Посылка">
          {order.description}
          <div className="text-muted-foreground">
            {order.weightKg} кг{order.declaredValue > 0 && ` · ценность ${money(order.declaredValue)}`}
          </div>
        </Info>
        <Info label="Доставка">
          {order.deliveryTitle}
          <div className="text-muted-foreground">{order.pickupPointName ?? `${order.address}`}</div>
        </Info>
        <Info label="Район и расстояние">
          {order.districtName}
          <div className="text-muted-foreground">{order.distanceKm} км от склада</div>
        </Info>
        <Info label="Курьер">{order.courierName ?? <span className="text-muted-foreground">не назначен</span>}</Info>
        <Info label="Доп. услуги">
          {extraTitles.length ? extraTitles.join(', ') : <span className="text-muted-foreground">нет</span>}
        </Info>
      </dl>

      <div className="rounded-lg bg-muted/60 p-3 text-sm">
        <div className="mb-2 flex justify-between font-medium">
          <span>Стоимость</span>
          <span className="text-muted-foreground">{strategyTitle}</span>
        </div>
        <ul className="grid gap-1">
          {order.priceLines.map((line) => (
            <li key={line.label} className="flex justify-between gap-3">
              <span className="text-muted-foreground">{line.label}</span>
              <span className="tabular-nums">{money(line.amount)}</span>
            </li>
          ))}
        </ul>
        <div className="mt-2 flex justify-between border-t pt-2 font-semibold">
          <span>Итого</span>
          <span className="tabular-nums">{money(order.price)}</span>
        </div>
      </div>

      <Actions>
        {order.assignable && (
          <div className="flex w-full flex-wrap items-center gap-2">
            <OptionSelect
              value={selector}
              onChange={setSelector}
              options={options.courierSelectors}
              className="w-56"
            />
            <Button disabled={busy} onClick={() => run(() => api.assignCourier(order.id, selector))}>
              Назначить курьера
            </Button>
          </div>
        )}
        {order.nextStatuses.map((s) => (
          <Button
            key={s.code}
            variant={s.code === 'cancelled' ? 'destructive' : 'secondary'}
            disabled={busy}
            onClick={() => run(() => api.changeStatus(order.id, s.code))}
          >
            {s.code === 'cancelled' ? 'Отменить заказ' : `→ ${s.title}`}
          </Button>
        ))}
        {order.editable && (
          <Button variant="outline" onClick={() => setEditing(true)}>
            Изменить
          </Button>
        )}
        {order.deletable && (
          <Button variant="ghost" className="text-destructive" onClick={() => setConfirmDelete(true)}>
            Удалить
          </Button>
        )}
      </Actions>

      <Separator />
      <div className="text-sm">
        <div className="mb-2 font-medium">История</div>
        <ol className="grid gap-1.5">
          {order.history.map((h, i) => (
            <li key={i} className="flex gap-3">
              <span className="w-24 shrink-0 tabular-nums text-muted-foreground">{dateTime(h.at)}</span>
              <span>{options.statuses.find((s) => s.code === h.to)?.title ?? h.to}</span>
            </li>
          ))}
        </ol>
      </div>

      <OrderFormDialog order={order} open={editing} onOpenChange={setEditing} />
      <DeleteOrderDialog
        order={order}
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        onDeleted={() => {
          toast.success(`Заказ ${order.id} удалён`)
          refresh()
          onDeleted()
        }}
      />
    </>
  )
}

function Info({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5">{children}</dd>
    </div>
  )
}

function Actions({ children }: { children: ReactNode }) {
  return <div className="flex flex-wrap gap-2">{children}</div>
}

function DeleteOrderDialog({
  order,
  open,
  onOpenChange,
  onDeleted,
}: {
  order: Order
  open: boolean
  onOpenChange: (open: boolean) => void
  onDeleted: () => void
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Удалить заказ {order.id}?</AlertDialogTitle>
          <AlertDialogDescription>Заказ пропадёт из списка и отчёта. Это нельзя отменить.</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Отмена</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() =>
              api
                .deleteOrder(order.id)
                .then(() => {
                  onOpenChange(false)
                  onDeleted()
                })
                .catch((e) => toast.error(errorMessage(e)))
            }
          >
            Удалить
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
