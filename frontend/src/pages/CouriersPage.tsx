import { useEffect, useState } from 'react'
import { PencilIcon, PlusIcon, Trash2Icon } from 'lucide-react'
import { toast } from 'sonner'
import { EmptyState, Field, LoadBar, OptionSelect } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError, api, errorMessage, type Courier } from '@/lib/api'
import { useApp, useLoad } from '@/lib/app'
import { ConfirmDelete } from './ConfirmDelete'

export function CouriersPage() {
  const { refresh } = useApp()
  const { data: couriers } = useLoad(api.couriers, [])
  const { data: points } = useLoad(api.pickupPoints, [])
  const [editing, setEditing] = useState<Courier | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Courier | null>(null)

  return (
    <div className="grid gap-4">
      <div className="flex justify-end">
        <Button onClick={() => setEditing('new')}>
          <PlusIcon /> Новый курьер
        </Button>
      </div>
      <Card className="py-0">
        <CardContent className="px-0">
          {couriers?.length === 0 ? (
            <EmptyState>Курьеров пока нет</EmptyState>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-4">Курьер</TableHead>
                  <TableHead>Телефон</TableHead>
                  <TableHead>Район</TableHead>
                  <TableHead>Загрузка</TableHead>
                  <TableHead className="w-24 pr-4" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {couriers?.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="pl-4 font-medium">{c.name}</TableCell>
                    <TableCell className="tabular-nums">{c.phone}</TableCell>
                    <TableCell>{c.districtName}</TableCell>
                    <TableCell>
                      <LoadBar value={c.activeOrders} max={c.capacity} />
                    </TableCell>
                    <TableCell className="pr-4 text-right">
                      <Button variant="ghost" size="icon" onClick={() => setEditing(c)} aria-label="Изменить">
                        <PencilIcon />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => setDeleting(c)} aria-label="Удалить">
                        <Trash2Icon />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Пункты выдачи и постаматы</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-2 sm:grid-cols-2">
          {points?.map((p) => (
            <div key={p.id} className="rounded-lg border p-3 text-sm">
              <div className="font-medium">{p.name}</div>
              <div className="text-muted-foreground">
                {p.address} · {p.districtName} · {p.kindTitle}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      <CourierDialog courier={editing} onClose={() => setEditing(null)} onSaved={refresh} />
      <ConfirmDelete
        title={`Удалить курьера «${deleting?.name}»?`}
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        onConfirm={() => api.deleteCourier(deleting!.id)}
        onDone={refresh}
      />
    </div>
  )
}

type CourierForm = { name: string; phone: string; district: string; capacity: string }

function CourierDialog({
  courier,
  onClose,
  onSaved,
}: {
  courier: Courier | 'new' | null
  onClose: () => void
  onSaved: () => void
}) {
  const { options } = useApp()
  const [form, setForm] = useState<CourierForm>({ name: '', phone: '', district: '', capacity: '3' })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const existing = courier !== 'new' ? courier : null

  useEffect(() => {
    if (courier) {
      setForm(
        existing
          ? { name: existing.name, phone: existing.phone, district: existing.district, capacity: String(existing.capacity) }
          : { name: '', phone: '', district: '', capacity: '3' },
      )
      setErrors({})
    }
  }, [courier, existing])

  async function submit() {
    const input = { ...form, capacity: Number(form.capacity) || 0 }
    try {
      if (existing) await api.updateCourier(existing.id, input)
      else await api.createCourier(input)
      toast.success('Курьер сохранён')
      onSaved()
      onClose()
    } catch (e) {
      if (e instanceof ApiError) setErrors(e.fields)
      toast.error(errorMessage(e))
    }
  }

  return (
    <Dialog open={courier !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{existing ? 'Изменить курьера' : 'Новый курьер'}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4">
          <Field label="Имя" error={errors.name}>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label="Телефон" error={errors.phone}>
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Район" error={errors.district}>
              <OptionSelect
                value={form.district}
                onChange={(v) => setForm({ ...form, district: v })}
                options={options?.districts ?? []}
                invalid={!!errors.district}
              />
            </Field>
            <Field label="Вместимость" error={errors.capacity} hint="заказов одновременно">
              <Input
                inputMode="numeric"
                value={form.capacity}
                onChange={(e) => setForm({ ...form, capacity: e.target.value })}
              />
            </Field>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Отмена
          </Button>
          <Button onClick={submit}>Сохранить</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
