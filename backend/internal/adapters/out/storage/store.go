package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// Store открывает все JSON-файлы в папке данных.
type Store struct {
	Orders       *Collection[*domain.Order]
	Customers    *Collection[*domain.Customer]
	Couriers     *Collection[*domain.Courier]
	PickupPoints *Collection[*domain.PickupPoint]
	Activity     *ActivityLog
	Exports      *ExportFolder
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{}
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
	if s.Activity, err = openActivityLog(filepath.Join(dir, "activity.json")); err != nil {
		return nil, err
	}
	s.Exports = &ExportFolder{dir: filepath.Join(dir, "exports")}
	return s, nil
}

// IsEmpty — true при самом первом запуске (тогда загружаются тестовые данные).
func (s *Store) IsEmpty() bool {
	return s.Orders.Len() == 0 && s.Customers.Len() == 0 && s.Couriers.Len() == 0 && s.PickupPoints.Len() == 0
}

// ActivityLog — журнал событий (новые записи в конце файла).
type ActivityLog struct {
	mu      sync.Mutex
	path    string
	entries []port.ActivityEntry
}

const maxActivityEntries = 1000

func openActivityLog(path string) (*ActivityLog, error) {
	log := &ActivityLog{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return log, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &log.entries); err != nil {
		return nil, fmt.Errorf("файл %s повреждён: %w", path, err)
	}
	return log, nil
}

func (l *ActivityLog) Append(entry port.ActivityEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.ID == "" {
		entry.ID = domain.NewID("EVT")
	}
	l.entries = append(l.entries, entry)
	if len(l.entries) > maxActivityEntries {
		l.entries = l.entries[len(l.entries)-maxActivityEntries:]
	}
	data, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(l.path, data)
}

// List отдаёт последние записи, самые новые первыми.
func (l *ActivityLog) List(limit int) ([]port.ActivityEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limit <= 0 || limit > len(l.entries) {
		limit = len(l.entries)
	}
	result := make([]port.ActivityEntry, 0, limit)
	for i := len(l.entries) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, l.entries[i])
	}
	return result, nil
}

// ExportFolder сохраняет выгруженные отчёты в data/exports.
type ExportFolder struct {
	dir string
}

func (e *ExportFolder) SaveExport(name string, data []byte) (string, error) {
	if err := os.MkdirAll(e.dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(e.dir, filepath.Base(name))
	return path, writeFileAtomic(path, data)
}
