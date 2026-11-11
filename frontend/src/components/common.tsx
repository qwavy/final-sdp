import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Option } from '@/lib/api'
import { cn } from '@/lib/utils'

// Field — подпись, поле ввода и текст ошибки под ним.
export function Field({
  label,
  error,
  hint,
  children,
  className,
}: {
  label: string
  error?: string
  hint?: string
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn('grid gap-1.5', className)}>
      <Label>{label}</Label>
      {children}
      {error ? (
        <p className="text-xs text-destructive">{error}</p>
      ) : (
        hint && <p className="text-xs text-muted-foreground">{hint}</p>
      )}
    </div>
  )
}

// OptionSelect — выпадающий список из пар {code, title}.
export function OptionSelect({
  value,
  onChange,
  options,
  placeholder = 'Выберите…',
  invalid,
  className,
}: {
  value: string
  onChange: (value: string) => void
  options: Option[]
  placeholder?: string
  invalid?: boolean
  className?: string
}) {
  return (
    <Select
      value={value || null}
      onValueChange={(v) => onChange((v as string | null) ?? '')}
      items={options.map((o) => ({ value: o.code, label: o.title }))}
    >
      <SelectTrigger className={cn('w-full', className)} aria-invalid={invalid || undefined}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.code} value={o.code}>
            {o.title}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

const statusStyles: Record<string, string> = {
  created: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200',
  assigned: 'bg-violet-100 text-violet-700 dark:bg-violet-950 dark:text-violet-300',
  in_transit: 'bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300',
  at_pickup_point: 'bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300',
  delivered: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300',
  cancelled: 'bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300',
}

export function StatusBadge({ status, title }: { status: string; title: string }) {
  return <Badge className={cn('border-0', statusStyles[status])}>{title}</Badge>
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <div className="py-12 text-center text-sm text-muted-foreground">{children}</div>
}

export function LoadBar({ value, max }: { value: number; max: number }) {
  const percent = max > 0 ? Math.min(100, (value / max) * 100) : 0
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div
          className={cn('h-full rounded-full', percent >= 100 ? 'bg-rose-500' : 'bg-primary')}
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className="text-xs tabular-nums text-muted-foreground">
        {value} / {max}
      </span>
    </div>
  )
}
