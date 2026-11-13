import { useEffect, useState } from 'react'
import { PencilIcon, PlusIcon, Trash2Icon } from 'lucide-react'
import { toast } from 'sonner'
import { EmptyState, Field } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ApiError, api, errorMessage, type Customer, type CustomerInput } from '@/lib/api'
import { useApp, useLoad } from '@/lib/app'
import { ConfirmDelete } from './ConfirmDelete'

export function CustomersPage() {
  const { refresh } = useApp()
  const { data: customers } = useLoad(api.customers, [])
  const [editing, setEditing] = useState<Customer | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Customer | null>(null)

  return (
    <div className="grid gap-4">
      <div className="flex justify-end">
        <Button onClick={() => setEditing('new')}>
          <PlusIcon /> Новый клиент
        </Button>
      </div>
      <Card className="py-0">
        <CardContent className="px-0">
          {customers?.length === 0 ? (
            <EmptyState>Клиентов пока нет</EmptyState>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-4">Имя</TableHead>
                  <TableHead>Телефон</TableHead>
                  <TableHead>Заказы (активные / всего)</TableHead>
                  <TableHead className="w-24 pr-4" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {customers?.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="pl-4 font-medium">{c.name}</TableCell>
                    <TableCell className="tabular-nums">{c.phone}</TableCell>
                    <TableCell className="tabular-nums">
                      {c.activeOrders} / {c.totalOrders}
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

      <CustomerDialog customer={editing} onClose={() => setEditing(null)} onSaved={refresh} />
      <ConfirmDelete
        title={`Удалить клиента «${deleting?.name}»?`}
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        onConfirm={() => api.deleteCustomer(deleting!.id)}
        onDone={refresh}
      />
    </div>
  )
}

function CustomerDialog({
  customer,
  onClose,
  onSaved,
}: {
  customer: Customer | 'new' | null
  onClose: () => void
  onSaved: () => void
}) {
  const [form, setForm] = useState<CustomerInput>({ name: '', phone: '' })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const existing = customer !== 'new' ? customer : null

  useEffect(() => {
    if (customer) {
      setForm(existing ? { name: existing.name, phone: existing.phone } : { name: '', phone: '' })
      setErrors({})
    }
  }, [customer, existing])

  async function submit() {
    try {
      if (existing) await api.updateCustomer(existing.id, form)
      else await api.createCustomer(form)
      toast.success('Клиент сохранён')
      onSaved()
      onClose()
    } catch (e) {
      if (e instanceof ApiError) setErrors(e.fields)
      toast.error(errorMessage(e))
    }
  }

  return (
    <Dialog open={customer !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{existing ? 'Изменить клиента' : 'Новый клиент'}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4">
          <Field label="Имя" error={errors.name}>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label="Телефон" error={errors.phone} hint="можно 8 701 123 45 67 — приведём к +7…">
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
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
