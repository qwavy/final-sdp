// Package storage — хранение данных в JSON-файлах (по файлу на тип сущности).
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"parcelhub/internal/domain"
)

// Collection — набор сущностей одного типа в одном JSON-файле.
// Внутри хранится JSON каждой сущности, поэтому Get/List всегда отдают свежую копию:
// изменения объекта не попадут в хранилище, пока не вызван Save.
type Collection[T any] struct {
	mu    sync.Mutex
	path  string
	what  string // «заказ», «клиент» — для текста ошибки
	idOf  func(T) string
	items map[string]json.RawMessage
}

func openCollection[T any](path, what string, idOf func(T) string) (*Collection[T], error) {
	c := &Collection[T]{path: path, what: what, idOf: idOf, items: map[string]json.RawMessage{}}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("файл %s повреждён: %w", path, err)
	}
	for _, raw := range list {
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("файл %s повреждён: %w", path, err)
		}
		c.items[idOf(item)] = raw
	}
	return c, nil
}

func (c *Collection[T]) List() ([]T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]T, 0, len(c.items))
	for _, id := range c.sortedIDs() {
		item, err := decode[T](c.items[id])
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (c *Collection[T]) Get(id string) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	raw, ok := c.items[id]
	if !ok {
		var zero T
		return zero, domain.NotFound("%s %s не найден", c.what, id)
	}
	return decode[T](raw)
}

func (c *Collection[T]) Save(item T) error {
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.idOf(item)
	previous, existed := c.items[id]
	c.items[id] = raw
	if err := c.flush(); err != nil {
		if existed { // откатываем изменение в памяти, раз на диск не записалось
			c.items[id] = previous
		} else {
			delete(c.items, id)
		}
		return err
	}
	return nil
}

func (c *Collection[T]) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	previous, ok := c.items[id]
	if !ok {
		return domain.NotFound("%s %s не найден", c.what, id)
	}
	delete(c.items, id)
	if err := c.flush(); err != nil {
		c.items[id] = previous
		return err
	}
	return nil
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

func (c *Collection[T]) flush() error {
	list := make([]json.RawMessage, 0, len(c.items))
	for _, id := range c.sortedIDs() {
		list = append(list, c.items[id])
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(c.path, data)
}

func decode[T any](raw json.RawMessage) (T, error) {
	var item T
	err := json.Unmarshal(raw, &item)
	return item, err
}

// writeFileAtomic пишет во временный файл и переименовывает его,
// чтобы при сбое не остался наполовину записанный JSON.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
