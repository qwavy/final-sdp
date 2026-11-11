import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Field, OptionSelect } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError, api, errorMessage, type Order, type OrderInput, type QuoteOption } from '@/lib/api'
import { money } from '@/lib/format'
import { useLive, useLoad } from '@/lib/live'
import { cn } from '@/lib/utils'

// Поля формы храним строками, чтобы пользователь мог стереть число целиком.
type FormState = Omit<OrderInput, 'weightKg' | 'declaredValue'> & { weightKg: string; declaredValue: string }

const emptyForm: FormState = {
  customerId: '',
  description: '',
  weightKg: '',
  declaredValue: '',
  deliveryType: 'door',
  address: '',
  district: '',
  pickupPointId: '',
  pricingStrategy: 'by_weight',
  extras: [],
}

function fromOrder(o: Order): FormState {
  return {
    customerId: o.customerId,
    description: o.description,
    weightKg: String(o.weightKg),
    declaredValue: o.declaredValue ? String(o.declaredValue) : '',
    deliveryType: o.deliveryType,
    address: o.address ?? '',
    district: o.district,
    pickupPointId: o.pickupPointId ?? '',
    pricingStrategy: o.pricingStrategy,
    extras: o.extras ?? [],
  }
}

function toInput(f: FormState): OrderInput {
  return { ...f, weightKg: Number(f.weightKg.replace(',', '.')) || 0, declaredValue: Number(f.declaredValue) || 0 }
}

export function OrderFormDialog({
  order,
  open,
  onOpenChange,
  onSaved,
}: {
  order?: Order
  open: boolean
  onOpenChange: (open: boolean) => void
  onSaved?: (order: Order) => void
}) {
  const { options } = useLive()
  const customers = useLoad(api.customers, [])
  const pickupPoints = useLoad(api.pickupPoints, [])

  const [form, setForm] = useState<FormState>(emptyForm)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [quotes, setQuotes] = useState<QuoteOption[]>([])
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (open) {
      setForm(order ? fromOrder(order) : emptyForm)
      setErrors({})
    }
  }, [open, order])

  // Предварительный расчёт цены по всем стратегиям (с задержкой, пока пользователь печатает).
  useEffect(() => {
    if (!open) return
    const timer = setTimeout(() => {
      api
        .quote(toInput(form))
        .then(setQuotes)
        .catch(() => setQuotes([]))
    }, 250)
    return () => clearTimeout(timer)
  }, [open, form])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((f) => ({ ...f, [key]: value }))
    setErrors((e) => ({ ...e, [key]: '' }))
  }

  const delivery = options?.deliveryTypes.find((d) => d.code === form.deliveryType)
  const points = (pickupPoints.data ?? []).filter((p) => p.kind === form.deliveryType)
  const selectedQuote = quotes.find((q) => q.strategy === form.pricingStrategy)

  async function submit() {
    setSaving(true)
    try {
      const saved = order ? await api.updateOrder(order.id, toInput(form)) : await api.createOrder(toInput(form))
      toast.success(order ? `Заказ ${saved.id} сохранён` : `Заказ ${saved.id} создан`)
      onOpenChange(false)
      onSaved?.(saved)
    } catch (e) {
      if (e instanceof ApiError && Object.keys(e.fields).length > 0) {
        setErrors(e.fields)
        toast.error(e.message)
      } else {
        toast.error(errorMessage(e))
      }
    } finally {
      setSaving(false)
    }
  }

  if (!options) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{order ? `Изменить заказ ${order.id}` : 'Новый заказ'}</DialogTitle>
        </DialogHeader>

        <div className="grid gap-6 md:grid-cols-[1fr_280px]">
          <div className="grid content-start gap-4">
            <Field label="Клиент" error={errors.customerId}>
              <OptionSelect
                value={form.customerId}
                onChange={(v) => set('customerId', v)}
                options={(customers.data ?? []).map((c) => ({ code: c.id, title: `${c.name} · ${c.phone}` }))}
                placeholder="Выберите клиента"
                invalid={!!errors.customerId}
              />
            </Field>

            <Field label="Что внутри" error={errors.description}>
              <Input
                value={form.description}
                onChange={(e) => set('description', e.target.value)}
                placeholder="Например: книги, 3 шт."
                aria-invalid={!!errors.description || undefined}
              />
            </Field>

            <div className="grid grid-cols-2 gap-4">
              <Field
                label="Вес, кг"
                error={errors.weightKg}
                hint={delivery ? `до ${delivery.maxWeightKg} кг для «${delivery.title}»` : undefined}
              >
                <Input
                  inputMode="decimal"
                  value={form.weightKg}
                  onChange={(e) => set('weightKg', e.target.value)}
                  placeholder="0.5"
                  aria-invalid={!!errors.weightKg || undefined}
                />
              </Field>
              <Field label="Объявленная ценность, ₸" error={errors.declaredValue} hint="нужна для страховки">
                <Input
                  inputMode="numeric"
                  value={form.declaredValue}
                  onChange={(e) => set('declaredValue', e.target.value)}
                  placeholder="0"
                  aria-invalid={!!errors.declaredValue || undefined}
                />
              </Field>
            </div>

            <Field label="Способ доставки" error={errors.deliveryType}>
              <div className="grid grid-cols-3 gap-2">
                {options.deliveryTypes.map((d) => (
                  <button
                    key={d.code}
                    type="button"
                    onClick={() => setForm((f) => ({ ...f, deliveryType: d.code, pickupPointId: '' }))}
                    className={cn(
                      'rounded-lg border p-2.5 text-left text-sm transition-colors hover:bg-muted',
                      form.deliveryType === d.code && 'border-primary bg-muted ring-1 ring-primary',
                    )}
                  >
                    <div className="font-medium">{d.title}</div>
                    <div className="text-xs text-muted-foreground">
                      тариф {money(d.baseFee)} · до {d.maxWeightKg} кг
                    </div>
                  </button>
                ))}
              </div>
            </Field>

            {delivery?.needsPickupPoint ? (
              <Field label={delivery.title === 'В постамат' ? 'Постамат' : 'Пункт выдачи'} error={errors.pickupPointId}>
                <OptionSelect
                  value={form.pickupPointId}
                  onChange={(v) => set('pickupPointId', v)}
                  options={points.map((p) => ({ code: p.id, title: `${p.name} — ${p.address} (${p.districtName})` }))}
                  placeholder="Выберите точку"
                  invalid={!!errors.pickupPointId}
                />
              </Field>
            ) : (
              <div className="grid grid-cols-[1fr_180px] gap-4">
                <Field label="Адрес" error={errors.address}>
                  <Input
                    value={form.address}
                    onChange={(e) => set('address', e.target.value)}
                    placeholder="Улица, дом, квартира"
                    aria-invalid={!!errors.address || undefined}
                  />
                </Field>
                <Field label="Район" error={errors.district}>
                  <OptionSelect
                    value={form.district}
                    onChange={(v) => set('district', v)}
                    options={options.districts}
                    placeholder="Район"
                    invalid={!!errors.district}
                  />
                </Field>
              </div>
            )}

            <div className="grid gap-2">
              <Label>Дополнительные услуги</Label>
              <div className="grid gap-2 sm:grid-cols-3">
                {options.extras.map((extra) => (
                  <label
                    key={extra.code}
                    className="flex cursor-pointer items-start gap-2 rounded-lg border p-2.5 text-sm hover:bg-muted"
                  >
                    <Checkbox
                      checked={form.extras.includes(extra.code)}
                      onCheckedChange={(checked) =>
                        set(
                          'extras',
                          checked ? [...form.extras, extra.code] : form.extras.filter((c) => c !== extra.code),
                        )
                      }
                    />
                    <span>
                      <span className="block font-medium">{extra.title}</span>
                      <span className="text-xs text-muted-foreground">{extra.hint}</span>
                    </span>
                  </label>
                ))}
              </div>
              {errors.extras && <p className="text-xs text-destructive">{errors.extras}</p>}
            </div>
          </div>

          <div className="grid content-start gap-3">
            <Label>Способ расчёта цены</Label>
            {options.pricingStrategies.map((s) => {
              const quote = quotes.find((q) => q.strategy === s.code)
              return (
                <button
                  key={s.code}
                  type="button"
                  onClick={() => set('pricingStrategy', s.code)}
                  className={cn(
                    'rounded-lg border p-3 text-left transition-colors hover:bg-muted',
                    form.pricingStrategy === s.code && 'border-primary bg-muted ring-1 ring-primary',
                  )}
                >
                  <div className="text-sm">{s.title}</div>
                  <div className="mt-1 text-xl font-semibold tabular-nums">{quote ? money(quote.total) : '—'}</div>
                </button>
              )
            })}
            {errors.pricingStrategy && <p className="text-xs text-destructive">{errors.pricingStrategy}</p>}

            <div className="rounded-lg bg-muted/60 p-3 text-sm">
              <div className="mb-2 font-medium">Расшифровка</div>
              {selectedQuote ? (
                <>
                  <ul className="grid gap-1.5">
                    {selectedQuote.lines.map((line) => (
                      <li key={line.label} className="flex justify-between gap-3">
                        <span className="text-muted-foreground">{line.label}</span>
                        <span className="shrink-0 tabular-nums">{money(line.amount)}</span>
                      </li>
                    ))}
                  </ul>
                  <div className="mt-2 flex justify-between border-t pt-2 font-semibold">
                    <span>Итого</span>
                    <span className="tabular-nums">{money(selectedQuote.total)}</span>
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    Расстояние от склада: {selectedQuote.distanceKm} км
                  </div>
                </>
              ) : (
                <p className="text-muted-foreground">Укажите вес и куда везти — цена посчитается сама.</p>
              )}
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button onClick={submit} disabled={saving}>
            {order ? 'Сохранить' : 'Создать заказ'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
