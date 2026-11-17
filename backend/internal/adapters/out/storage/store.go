package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// Store — все данные приложения: по JSON-файлу на тип сущности.
type Store struct {
	Orders       *Collection[domain.Order]
	Customers    *Collection[domain.Customer]
	Couriers     *Collection[domain.Courier]
	PickupPoints *Collection[domain.PickupPoint]
	Activity     *ActivityLog
	Exports      *ExportFolder
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{
		Exports: &ExportFolder{dir: filepath.Join(dir, "exports")},
	}
	var err error
	if s.Orders, err = openCollection(filepath.Join(dir, "orders.json"), "заказ",
		func(o *domain.Order) string { return o.ID }); err != nil {
		return nil, err
	}
	if s.Customers, err = openCollection(filepath.Join(dir, "customers.json"), "клиент",
		func(c *domain.Customer) string { return c.ID }); err != nil {
		return nil, err
	}
	if s.Couriers, err = openCollection(filepath.Join(dir, "couriers.json"), "курьер",
		func(c *domain.Courier) string { return c.ID }); err != nil {
		return nil, err
	}
	if s.PickupPoints, err = openCollection(filepath.Join(dir, "pickup_points.json"), "пункт выдачи",
		func(p *domain.PickupPoint) string { return p.ID }); err != nil {
		return nil, err
	}
	s.Activity = &ActivityLog{path: filepath.Join(dir, "activity.json")}
	if data, err := os.ReadFile(s.Activity.path); err == nil {
		_ = json.Unmarshal(data, &s.Activity.entries)
	}
	return s, nil
}

// IsEmpty — true при первом запуске (тогда загружаются тестовые данные).
func (s *Store) IsEmpty() bool {
	return s.Customers.Len() == 0 && s.Couriers.Len() == 0
}

// ActivityLog — журнал событий, хранится в activity.json.
type ActivityLog struct {
	mu      sync.Mutex
	path    string
	entries []port.ActivityEntry
}

func (l *ActivityLog) Append(entry port.ActivityEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.ID = domain.NewID("EVT")
	l.entries = append(l.entries, entry)
	data, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, data, 0o644)
}

// List — последние записи, новые первыми.
func (l *ActivityLog) List(limit int) ([]port.ActivityEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := slices.Clone(l.entries)
	slices.Reverse(result)
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

// ExportFolder сохраняет выгруженные отчёты в data/exports.
type ExportFolder struct {
	dir string
}

func (e *ExportFolder) SaveExport(name string, data []byte) error {
	if err := os.MkdirAll(e.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.dir, name), data, 0o644)
}
