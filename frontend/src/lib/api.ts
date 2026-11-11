// Клиент REST API Go-сервера. Все типы повторяют JSON-ответы бэкенда.

export type Option = { code: string; title: string }
export type DeliveryOption = Option & { baseFee: number; maxWeightKg: number; needsPickupPoint: boolean }
export type ExtraOption = Option & { hint: string }

export type Options = {
  deliveryTypes: DeliveryOption[]
  pricingStrategies: Option[]
  courierSelectors: Option[]
  extras: ExtraOption[]
  districts: Option[]
  statuses: Option[]
}

export type PriceLine = { label: string; amount: number }
export type StatusChange = { from?: string; to: string; at: string }

export type Order = {
  id: string
  customerId: string
  description: string
  weightKg: number
  declaredValue: number
  deliveryType: string
  address?: string
  pickupPointId?: string
  district: string
  distanceKm: number
  pricingStrategy: string
  extras: string[]
  priceLines: PriceLine[]
  price: number
  courierId?: string
  status: string
  history: StatusChange[]
  createdAt: string
  updatedAt: string
  customerName: string
  customerPhone: string
  courierName?: string
  pickupPointName?: string
  districtName: string
  deliveryTitle: string
  statusTitle: string
  nextStatuses: Option[]
  assignable: boolean
  editable: boolean
  deletable: boolean
}

export type OrderInput = {
  customerId: string
  description: string
  weightKg: number
  declaredValue: number
  deliveryType: string
  address: string
  district: string
  pickupPointId: string
  pricingStrategy: string
  extras: string[]
}

export type QuoteOption = {
  strategy: string
  title: string
  distanceKm: number
  total: number
  lines: PriceLine[]
}

export type Customer = {
  id: string
  name: string
  phone: string
  totalOrders: number
  activeOrders: number
}
export type CustomerInput = { name: string; phone: string }

export type Courier = {
  id: string
  name: string
  phone: string
  district: string
  districtName: string
  capacity: number
  activeOrders: number
}
export type CourierInput = { name: string; phone: string; district: string; capacity: number }

export type PickupPoint = {
  id: string
  name: string
  address: string
  district: string
  districtName: string
  kind: string
  kindTitle: string
}

export type ActivityEntry = { id: string; at: string; kind: string; orderId?: string; message: string }

export type DailyReport = {
  date: string
  totalOrders: number
  activeOrders: number
  deliveredOrders: number
  cancelledOrders: number
  revenue: number
  averagePrice: number
  byStatus: { status: string; title: string; count: number }[]
  byDelivery: { type: string; title: string; count: number; revenue: number }[]
  couriers: { id: string; name: string; activeOrders: number; capacity: number; delivered: number }[]
  orders: { id: string; created: string; customer: string; delivery: string; status: string; courier: string; price: number }[] | null
}

// ApiError — ошибка от сервера. fields — ошибки по полям формы (код 422).
export class ApiError extends Error {
  status: number
  fields: Record<string, string>

  constructor(status: number, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.status = status
    this.fields = fields
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'Сервер недоступен. Запущен ли backend?')
  }

  if (response.status === 204) return undefined as T
  const data = await response.json().catch(() => null)
  if (!response.ok) {
    const fields: Record<string, string> = {}
    for (const f of data?.fields ?? []) fields[f.field] ??= f.message
    throw new ApiError(response.status, data?.error ?? `Ошибка ${response.status}`, fields)
  }
  return data as T
}

export const api = {
  options: () => request<Options>('GET', '/api/options'),

  orders: (status: string, q: string) =>
    request<Order[]>('GET', `/api/orders?${new URLSearchParams({ status, q })}`),
  order: (id: string) => request<Order>('GET', `/api/orders/${id}`),
  quote: (input: Partial<OrderInput>) => request<QuoteOption[]>('POST', '/api/orders/quote', input),
  createOrder: (input: OrderInput) => request<Order>('POST', '/api/orders', input),
  updateOrder: (id: string, input: OrderInput) => request<Order>('PUT', `/api/orders/${id}`, input),
  deleteOrder: (id: string) => request<void>('DELETE', `/api/orders/${id}`),
  assignCourier: (id: string, selector: string) =>
    request<Order>('POST', `/api/orders/${id}/assign`, { selector }),
  changeStatus: (id: string, status: string) => request<Order>('POST', `/api/orders/${id}/status`, { status }),

  customers: () => request<Customer[]>('GET', '/api/customers'),
  createCustomer: (input: CustomerInput) => request<Customer>('POST', '/api/customers', input),
  updateCustomer: (id: string, input: CustomerInput) => request<Customer>('PUT', `/api/customers/${id}`, input),
  deleteCustomer: (id: string) => request<void>('DELETE', `/api/customers/${id}`),

  couriers: () => request<Courier[]>('GET', '/api/couriers'),
  createCourier: (input: CourierInput) => request<Courier>('POST', '/api/couriers', input),
  updateCourier: (id: string, input: CourierInput) => request<Courier>('PUT', `/api/couriers/${id}`, input),
  deleteCourier: (id: string) => request<void>('DELETE', `/api/couriers/${id}`),

  pickupPoints: () => request<PickupPoint[]>('GET', '/api/pickup-points'),
  activity: (limit = 200) => request<ActivityEntry[]>('GET', `/api/activity?limit=${limit}`),
  dailyReport: (date: string) => request<DailyReport>('GET', `/api/reports/daily?date=${date}`),
  exportUrl: (date: string, format: 'csv' | 'json') => `/api/reports/daily/export?date=${date}&format=${format}`,
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Неизвестная ошибка'
}
