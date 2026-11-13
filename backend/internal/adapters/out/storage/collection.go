// Package storage — хранение данных в JSON-файлах (по файлу на тип сущности).
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"

	"parcelhub/internal/domain"
)

// Collection — все объекты одного типа: в памяти — map, на диске — JSON-файл.
// После каждого изменения файл перезаписывается целиком.
type Collection[T any] struct {
	mu    sync.Mutex // сервер обрабатывает запросы параллельно
	path  string
	what  string // «заказ», «клиент» — для текста ошибки
	idOf  func(*T) string
	items map[string]T
}

// openCollection читает файл, если он есть. Если файла нет — коллекция пустая.
func openCollection[T any](path, what string, idOf func(*T) string) (*Collection[T], error) {
	c := &Collection[T]{path: path, what: what, idOf: idOf, items: map[string]T{}}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	var list []T
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("файл %s повреждён: %w", path, err)
	}
	for i := range list {
		c.items[idOf(&list[i])] = list[i]
	}
	return c, nil
}

// List возвращает копии всех объектов, отсортированные по ID.
func (c *Collection[T]) List() ([]*T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]*T, 0, len(c.items))
	for _, id := range c.sortedIDs() {
		item := c.items[id] // копия значения
		result = append(result, &item)
	}
	return result, nil
}

// Get возвращает копию объекта: изменения попадут в хранилище только через Save.
func (c *Collection[T]) Get(id string) (*T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[id]
	if !ok {
		return nil, domain.NotFound("%s %s не найден", c.what, id)
	}
	return &item, nil
}

func (c *Collection[T]) Save(item *T) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[c.idOf(item)] = *item
	return c.writeFile()
}

func (c *Collection[T]) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[id]; !ok {
		return domain.NotFound("%s %s не найден", c.what, id)
	}
	delete(c.items, id)
	return c.writeFile()
}

func (c *Collection[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

func (c *Collection[T]) sortedIDs() []string {
	ids := make([]string, 0, len(c.items))
	for id := range c.items {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (c *Collection[T]) writeFile() error {
	list := make([]T, 0, len(c.items))
	for _, id := range c.sortedIDs() {
		list = append(list, c.items[id])
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o644)
}
