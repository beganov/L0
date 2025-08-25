package cache

import (
	"testing"

	"github.com/beganov/L0/internal/models"
)

// вспомогательная функция для создания тестового заказа
func newTestOrder(id string) models.Order {
	return models.Order{
		OrderUID: id,
		Delivery: models.Delivery{Name: "John"},
		Payment:  models.Payment{Transaction: "txn1"},
		Items:    []models.Items{{ChrtID: 1, Name: "item1"}},
	}
}

// --- Тесты Set + Get ---
func TestOrderCache_SetAndGet(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		keys      []string
		expectKey string
		expectHit bool
	}{
		{
			name:      "simple set and get",
			capacity:  2,
			keys:      []string{"a"},
			expectKey: "a",
			expectHit: true,
		},
		{
			name:      "cache overflow evicts oldest",
			capacity:  2,
			keys:      []string{"a", "b", "c"},
			expectKey: "a",
			expectHit: false,
		},
		{
			name:      "cache hit on most recent",
			capacity:  2,
			keys:      []string{"a", "b"},
			expectKey: "b",
			expectHit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewOrderCache(tt.capacity)
			for _, k := range tt.keys {
				cache.Set(k, newTestOrder(k))
			}

			order, ok := cache.Get(tt.expectKey)
			if ok != tt.expectHit {
				t.Errorf("expected hit=%v, got %v", tt.expectHit, ok)
			}
			if ok && order.OrderUID != tt.expectKey {
				t.Errorf("expected OrderUID=%v, got %v", tt.expectKey, order.OrderUID)
			}
		})
	}
}

// --- Тест LRU-поведения ---
func TestOrderCache_LRUBehavior(t *testing.T) {
	cache := NewOrderCache(2)

	cache.Set("a", newTestOrder("a"))
	cache.Set("b", newTestOrder("b"))

	// "a" теперь в голове
	cache.Get("a")

	// добавляем "c", должно удалить "b"
	cache.Set("c", newTestOrder("c"))

	if _, ok := cache.Get("b"); ok {
		t.Errorf("expected 'b' to be evicted")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Errorf("expected 'a' to remain")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Errorf("expected 'c' to remain")
	}
}

// --- Тест для capacity = 1 ---
func TestOrderCache_CapacityOne(t *testing.T) {
	cache := NewOrderCache(1)

	cache.Set("x", newTestOrder("x"))
	cache.Set("y", newTestOrder("y"))

	if _, ok := cache.Get("x"); ok {
		t.Errorf("expected 'x' to be evicted")
	}
	if _, ok := cache.Get("y"); !ok {
		t.Errorf("expected 'y' to remain")
	}
}

// --- Тест на повторный Set ---
func TestOrderCache_SetExistingKey(t *testing.T) {
	cache := NewOrderCache(2)

	cache.Set("a", newTestOrder("a"))
	cache.Set("a", newTestOrder("a_new")) // должен проигнорировать

	order, ok := cache.Get("a")
	if !ok {
		t.Errorf("expected 'a' to exist")
	}
	if order.OrderUID != "a" {
		t.Errorf("expected OrderUID='a', got %v", order.OrderUID)
	}
}
