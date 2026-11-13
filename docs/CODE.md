# Parcel Hub — разбор кода

Этот файл объясняет, **как устроен код**: что за что отвечает и как проходит запрос.
Что сделано по ТЗ и зачем нужны паттерны — в [PROJECT.md](PROJECT.md).

---

## 1. Карта папок

```
backend/
  cmd/server/
    main.go          ← запуск: собирает все части вместе
    seed.go          ← тестовые данные при первом запуске
  internal/
    domain/          ← бизнес-логика и правила (ни от чего не зависит)
    application/
      facade.go      ← Facade: единственный вход для HTTP
      port/          ← интерфейсы «что нам нужно снаружи»
      events/        ← Observer: Publisher и наблюдатели
      use_cases/
        command/     ← команды: меняют данные
        query/       ← запросы: только читают
    adapters/
      in/http/       ← REST API
      out/storage/   ← хранение в JSON-файлах
      out/notify/    ← SMS-шлюз (Adapter)
frontend/src/
  lib/api.ts         ← все запросы к серверу
  lib/app.tsx        ← общее состояние: справочники и обновление таблиц
  pages/             ← экраны: заказы, клиенты, курьеры, журнал, отчёт
  components/ui/     ← готовые компоненты shadcn
```

**Правило зависимостей:** `adapters → application → domain`.
Внутренний слой никогда не импортирует внешний. Например, в `domain` нет ни одного `import`
из `application` или `adapters`.

`internal/` — особая папка Go: её код нельзя импортировать из других модулей.

---

## 2. Как проходит один запрос (на примере «сменить статус»)

Пользователь нажимает «→ В пути» в карточке заказа.

```
Браузер                         Сервер
───────                         ──────
api.changeStatus(id, "in_transit")
  POST /api/orders/{id}/status ──►  router.go: маршрут → handlers.changeStatus
                                     │ читает JSON {"status": "in_transit"}
                                     ▼
                                  DeliveryFacade.ChangeOrderStatus(id, status)
                                     │
                                     ├─► command.OrderCommands.ChangeStatus
                                     │     1. orders.Get(id)           ← из JSON-файла
                                     │     2. order.ChangeStatus(...)   ← domain проверяет правило
                                     │     3. orders.Save(order)        ← в JSON-файл
                                     │     4. publisher.Notify(event)   ← Observer
                                     │           ├─ ActivityRecorder → запись в журнал
                                     │           └─ CustomerNotifier → SMS через адаптер
                                     │
                                     └─► query.OrderQueries.Get(id)
                                           собирает OrderView с именами
  ◄──────────── 200 OK + JSON заказа ──────┘
  → уведомление «Заказ …: В пути»
  → refresh() → все таблицы перезагружают данные
```

Если правило нарушено (например, «Создан» → «Выдан»), на шаге 2 domain вернёт ошибку,
ничего не сохранится, и HTTP ответит `409` с понятным текстом.

---

## 3. Domain — `internal/domain`

| Файл | Что внутри |
|---|---|
| `order.go` | `Order`, статусы, методы `AssignCourier`, `ChangeStatus`, `ApplyEdit` |
| `order_builder.go` | **Builder**: `OrderBuilder` и вся проверка полей заказа |
| `delivery.go` | **Factory Method**: способы доставки и их правила статусов |
| `pricing.go` | **Strategy**: способы расчёта цены |
| `price.go` | **Decorator**: базовая цена + доп. услуги; функция `BuildPrice` |
| `courier_selection.go` | **Strategy**: способы выбора курьера |
| `customer.go`, `courier.go`, `pickup_point.go` | остальные сущности и их проверки |
| `errors.go` | виды ошибок: `ValidationError`, `NotFound`, `RuleViolation`, `InvalidTransition` |
| `geo.go` | районы, координаты, расстояние от склада |
| `common.go` | `Money`, генерация ID, нормализация телефона |

### Order

Заказ сам меняет своё состояние и проверяет правила. Например, смена статуса:

```go
func (o *Order) moveTo(to OrderStatus, now time.Time) {
    from := o.Status
    o.Status = to
    o.UpdatedAt = now
    o.History = append(o.History, StatusChange{From: from, To: to, At: now})
}
```

Domain не знает ни о журнале, ни об SMS — об этом заботится команда (см. Observer ниже).

### Как проверяются переходы статусов

Заказ сам не знает, куда можно перейти. Он спрашивает свой способ доставки:

```go
func (o *Order) NextStatuses() []OrderStatus {
    method, _ := NewDeliveryMethod(o.DeliveryType) // Factory Method
    return method.NextStatuses(o.Status)
}
```

У доставки до двери таблица `doorFlow`, у ПВЗ и постамата — `pickupFlow` (в `delivery.go`).

### Builder

```go
order, err := domain.NewOrderBuilder().
    ForCustomer("CUS-1").
    Parcel("Книги", 2.5, 10000).
    Delivery(domain.DeliveryLocker).
    ToPickupPoint(point).
    PricedBy("by_weight").
    WithExtras("fragile").
    Build(time.Now())
```

Каждый метод только запоминает значение и возвращает `b` — поэтому их можно вызывать цепочкой.
Вся проверка — в `Build()`: собирает **все** ошибки в `ValidationError` и возвращает их
вместе (форма подсвечивает все поля сразу). Если ошибок нет — считает расстояние, цену,
ставит статус «Создан».

`Invalid(field, msg)` — чтобы снаружи добавить ошибку, которую домен проверить не может
(например, «клиент не найден» — для этого нужно хранилище).

### Factory Method

```go
type DeliveryMethod interface {        // Product
    Type() DeliveryType
    BaseFee() Money
    MaxWeightKg() float64
    NeedsPickupPoint() bool
    NextStatuses(from OrderStatus) []OrderStatus
}

type DeliveryCreator interface {       // Creator
    CreateMethod() DeliveryMethod      // фабричный метод
}

type LockerDeliveryCreator struct{}    // ConcreteCreator
func (LockerDeliveryCreator) CreateMethod() DeliveryMethod { return lockerDelivery{} }
```

`NewDeliveryMethod(t)` берёт нужного создателя из карты `deliveryCreators` и вызывает его.

### Strategy

```go
type PricingStrategy interface {
    Code() string
    Title() string
    Calculate(in PricingInput) Money
}
```

`WeightPricing` и `DistancePricing` реализуют один интерфейс. Список `pricingStrategies`
— это «реестр»: по нему строится выбор в форме и ищется стратегия по коду.
`CourierSelector` устроен так же.

### Decorator

```go
type PriceComponent interface {
    Total() Money
    Lines() []PriceLine
}

type fragileExtra struct{ inner PriceComponent }   // обёртка хранит то, что оборачивает

func (e fragileExtra) Total() Money { return e.inner.Total() + 400 }
```

`BuildPrice` создаёт `basePrice`, а затем по очереди оборачивает его выбранными услугами:

```go
var price PriceComponent = basePrice{...}
for _, extra := range extras {
    if slices.Contains(spec.Extras, extra.Code) {
        price = extra.wrap(price, spec.DeclaredValue)  // обёртка вокруг предыдущей цены
    }
}
```

Получается «матрёшка»: `express( fragile( insurance( base ) ) )`.
Вызов `Total()` у внешней обёртки проходит вглубь до базы и возвращается, добавляя по пути
стоимость каждой услуги.

### Ошибки

- `ValidationError` — ошибки полей формы → HTTP `422`.
- `NotFound(...)` → `404`.
- `RuleViolation(...)`, `InvalidTransition(...)` → `409`.

HTTP-слой узнаёт вид ошибки через `errors.Is` / `errors.As`, а текст берёт из `err.Error()`.

---

## 4. Application — `internal/application`

### port — интерфейсы

```go
type Repository[T any] interface {
    List() ([]T, error)
    Get(id string) (T, error)
    Save(item T) error
    Delete(id string) error
}
type OrderRepository = Repository[*domain.Order]

type Notifier interface { Notify(n Notification) error }
```

Это **порты**: application говорит, что ему нужно, но не знает, кто это сделает.
Реализации — в `adapters/out`. `[T any]` — дженерик: один интерфейс для заказов,
клиентов, курьеров.

### events — Observer

```go
type Event struct {                     // что произошло
    Type    Type                        // "order.status_changed" и т.д.
    OrderID string
    Message string
    At      time.Time
}

type Observer interface {               // Observer
    Update(e Event)
}

type Publisher struct {                 // Subject
    observers []Observer
}

func (p *Publisher) Subscribe(o Observer) { p.observers = append(p.observers, o) }

func (p *Publisher) Notify(e Event) {
    for _, o := range p.observers {
        o.Update(e)
    }
}
```

- **Как работает:** `Publisher` хранит список наблюдателей. Команда после изменения заказа
  вызывает `Notify`, и каждый наблюдатель получает событие. Команда не знает,
  сколько наблюдателей и что они делают.
- **Наблюдатели** (`subscribers.go`):
  - `ActivityRecorder` — пишет каждое событие в журнал.
  - `CustomerNotifier` — реагирует только на `OrderStatusChanged`, находит клиента
    и отправляет SMS через `port.Notifier`. Результат (успех или ошибка) тоже пишет в журнал.
- **Где подписываются:** `cmd/server/main.go`, один раз при запуске.

### use_cases/command — команды

`OrderCommands` (`order.go`): `Create`, `Update`, `Delete`, `AssignCourier`, `ChangeStatus`.
`CustomerCommands`, `CourierCommands` (`directory.go`): добавление, изменение, удаление.

Общий шаблон каждой команды:
1. достать объект из репозитория;
2. вызвать метод домена (он проверит правила);
3. сохранить;
4. оповестить наблюдателей (`c.notify(...)` → `publisher.Notify`).

```go
func (c *OrderCommands) AssignCourier(id, selectorCode string) error {
    order, _ := c.orders.Get(id)
    selector, _ := domain.CourierSelectorByCode(selectorCode) // Strategy по выбору пользователя
    couriers, _ := c.couriers.List()
    courier, _ := selector.Select(order, couriers)
    order.AssignCourier(courier, c.now())                     // правило в домене
    c.couriers.Save(courier)
    c.orders.Save(order)
    c.notify(events.OrderStatusChanged, id, "...")            // Observer
    return nil
}
```
(в настоящем коде после каждого шага проверяется `err`.)

`now func() time.Time` передаётся снаружи, а не вызывается `time.Now()` напрямую —
чтобы в тестах можно было подставить фиксированное время.

### use_cases/query — запросы

- `OrderQueries.List / Get` — возвращают `OrderView`: заказ плюс имя клиента, курьера,
  название точки, список доступных статусов, флаги `editable / deletable / assignable`.
  Интерфейс просто показывает то, что пришло, и не дублирует бизнес-правила.
- `OrderQueries.Quote` — считает цену **по всем стратегиям** без сохранения (для формы).
- `DirectoryQueries` — клиенты, курьеры, точки, журнал, справочники для форм (`CatalogOptions`).
- `ReportQueries.Daily` — отчёт за день; `DailyReport.CSV()` / `.JSON()` — выгрузка.

`loadReferences` один раз загружает клиентов, курьеров и точки в карты `map[id]...`,
чтобы быстро подставлять имена вместо ID.

### facade.go — Facade

```go
func (f *DeliveryFacade) ChangeOrderStatus(orderID, status string) (query.OrderView, error) {
    if err := f.orderCmd.ChangeStatus(orderID, status); err != nil {   // команда
        return query.OrderView{}, err
    }
    return f.orders.Get(orderID)                                       // запрос
}
```

Фасад соединяет CQS-команду и запрос, чтобы HTTP получил готовый ответ одним вызовом.
`NewDeliveryFacade(Deps)` создаёт все команды и запросы — снаружи их никто не собирает.

---

## 5. Adapters — `internal/adapters`

### in/http — входной адаптер

| Файл | Что делает |
|---|---|
| `router.go` | список маршрутов (`mux.Handle("POST /api/orders", ...)`), раздача собранного фронтенда, лог запросов |
| `handlers.go` | обработчики: прочитать запрос → вызвать фасад → записать ответ |
| `respond.go` | `writeJSON`, `readJSON`, `writeError` (выбор HTTP-кода по виду ошибки) |

- **Маршруты** вида `"GET /api/orders/{id}"` — встроенная возможность `net/http` с Go 1.22.
  `r.PathValue("id")` достаёт `{id}` из пути.
- **`apiFunc`** — обработчик, который возвращает `error`. Если ошибка есть, `ServeHTTP`
  сам вызывает `writeError`. Так в обработчиках нет повторяющегося кода ответа об ошибке.
- **`spaHandler`** — отдаёт файлы из `frontend/dist`, а на неизвестные пути — `index.html`.

### out/storage — хранилище

- `Collection[T]` (`collection.go`) — один JSON-файл = один тип сущностей.
  В памяти — обычная `map[id]T`. При запуске файл читается, после каждого `Save` / `Delete`
  перезаписывается целиком.
- `Get` и `List` возвращают **копии**: если изменить объект и не вызвать `Save`,
  в хранилище ничего не поменяется.
- `sync.Mutex` — сервер обрабатывает запросы параллельно, мьютекс не даёт двум запросам
  одновременно менять `map`.
- `Store` (`store.go`) открывает все файлы: `orders.json`, `customers.json`, `couriers.json`,
  `pickup_points.json`, `activity.json`; `ExportFolder` сохраняет отчёты в `exports/`.

`Collection[T]` подходит под интерфейс `port.Repository[T]` — Go проверяет это сам,
явно писать «implements» не нужно.

### out/notify — Adapter

```go
// Adaptee: чужой SDK, его не меняем
func (g *SmsGatewayClient) SendSMS(msisdn, sender, body string) (statusCode int, messageID string)

// Adapter: реализует наш port.Notifier
func (a *SmsNotifier) Notify(n port.Notification) error {
    msisdn := strings.TrimPrefix(n.Phone, "+")       // +7701... → 7701...
    body := truncate(n.Text, 160)                    // лимит шлюза
    code, _ := a.gateway.SendSMS(msisdn, a.sender, body)
    switch code {                                    // код → error
    case GatewayOK:
        return nil
    ...
    }
}
```

`var _ port.Notifier = (*SmsNotifier)(nil)` — проверка при компиляции, что адаптер
действительно реализует интерфейс.

---

## 6. main.go — сборка (composition root)

```go
store := storage.Open(dataDir)                       // 1. хранилище
publisher := events.NewPublisher()                   // 2. Observer
sms := notify.NewSmsNotifier(...)                    // 3. адаптер SMS
publisher.Subscribe(events.NewActivityRecorder(...)) // 4. наблюдатели
publisher.Subscribe(events.NewCustomerNotifier(..., sms, ...))
facade := application.NewDeliveryFacade(Deps{...})  // 5. фасад
if store.IsEmpty() { seed(facade, store) }           // 6. тестовые данные
http.Server{Handler: httpadapter.NewRouter(...)}     // 7. HTTP
```

Это единственное место, где все слои встречаются. Остальной код получает зависимости
через параметры (dependency injection), поэтому в тестах их легко заменить.

`signal.NotifyContext` + `server.Shutdown` — аккуратная остановка по Ctrl+C.

---

## 7. Фронтенд — `frontend/src`

| Файл | Что делает |
|---|---|
| `lib/api.ts` | типы данных (как в JSON сервера) и функции запросов; `ApiError` с ошибками по полям |
| `lib/app.tsx` | `AppProvider`: загружает справочники для форм и хранит счётчик `version`. После любого действия вызывается `refresh()`, и хук `useLoad` заново загружает данные во всех таблицах |
| `pages/orders/OrdersPage.tsx` | таблица заказов, поиск, фильтр по статусу |
| `pages/orders/OrderForm.tsx` | форма заказа; живой расчёт цены через `/api/orders/quote` (с задержкой 250 мс) |
| `pages/orders/OrderDetails.tsx` | карточка заказа: цена, история, кнопки действий |
| `pages/CustomersPage.tsx`, `CouriersPage.tsx` | справочники с добавлением, изменением, удалением |
| `pages/ActivityPage.tsx`, `ReportPage.tsx` | журнал и отчёт с выгрузкой |
| `components/common.tsx` | общие куски: поле с ошибкой, выпадающий список, бейдж статуса |

Фронтенд **не содержит бизнес-правил**. Какие кнопки показать, решает сервер
(`nextStatuses`, `editable`, `deletable`), цену тоже считает сервер.

---

## 8. Тесты

```bash
cd backend && go test ./... -v
```

| Файл | Что проверяет |
|---|---|
| `domain/domain_test.go` | Builder (корректный и 6 неверных случаев), Strategy цены, Decorator, переходы статусов, Strategy курьера |
| `application/facade_test.go` | полный сценарий + Observer (с поддельным `fakeNotifier`), сохранение и загрузка, бизнес-правила |
| `adapters/out/notify/sms_test.go` | Adapter: формат номера, обрезка текста, код ошибки → `error` |

`t.TempDir()` создаёт временную папку для данных — тесты не трогают настоящий `backend/data`.
`fakeNotifier` подставляется вместо SMS — так видно пользу порта `Notifier`.

---

## 9. Типичные правки на защите — что менять

| Попросят | Что сделать |
|---|---|
| **Новая стратегия цены** (например, «фиксированная 1500 ₸») | В `domain/pricing.go`: новый тип с `Code`, `Title`, `Calculate` → добавить в `pricingStrategies`. В форме появится сам. |
| **Новая стратегия курьера** (например, «по алфавиту») | В `domain/courier_selection.go`: тип с `Select` → добавить в `courierSelectors`. |
| **Новая доп. услуга** (например, «подарочная упаковка +300 ₸») | В `domain/price.go`: тип-обёртка с `inner`, `Total`, `Lines` → запись в `extras` (до `express`). |
| **Новый способ доставки** (например, «самовывоз со склада») | В `domain/delivery.go`: новый продукт + новый `Creator` → добавить в `deliveryTypes` и `deliveryCreators`. |
| **Новый наблюдатель** (например, «писать в консоль») | Тип с методом `Update(e events.Event)` → `publisher.Subscribe(...)` в `main.go`. |
| **Новый канал уведомлений** (email) | В `adapters/out/notify`: тип с `Notify(port.Notification) error` → передать его вместо `sms` в `main.go`. |
| **Новое правило** (например, «вес не больше 40 кг для двери») | Поменять `MaxWeightKg()` у `doorDelivery` в `delivery.go`. |
| **Новое поле в форме** | `domain.Order` + `OrderBuilder` → `command.OrderInput` → `frontend/src/lib/api.ts` → `OrderForm.tsx`. |

После любой правки: `go test ./...` и перезапуск сервера.
