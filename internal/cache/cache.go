package cache

import (
	"sync"

	"github.com/beganov/L0/internal/metrics"
	"github.com/beganov/L0/internal/models"
)

// node in LRU list
type lruNode struct {
	key   string
	value models.Order
	prev  *lruNode
	next  *lruNode
}

// simple LRU cache for orders
type OrderCache struct {
	capacity int
	store    map[string]*lruNode
	head     *lruNode
	tail     *lruNode
	mu       sync.Mutex
}

// constructor
func NewOrderCache(cap int) *OrderCache {
	return &OrderCache{
		capacity: cap,
		store:    make(map[string]*lruNode),
	}
}

// move node to front (most recently used)
func (c *OrderCache) moveToFront(node *lruNode) {
	if c.head == node {
		return
	}

	// unlink node
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if c.tail == node {
		c.tail = node.prev
	}

	// put node at head
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node

	if c.tail == nil {
		c.tail = node
	}
}

// add new order to cache
func (c *OrderCache) Set(key string, order models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// already exists, skip
	if _, ok := c.store[key]; ok {
		return
	}

	node := &lruNode{key: key, value: order}
	c.store[key] = node
	c.moveToFront(node)

	// remove LRU if over capacity
	if len(c.store) > c.capacity {
		delete(c.store, c.tail.key)
		if c.tail.prev != nil {
			c.tail = c.tail.prev
			c.tail.next = nil
		} else {
			// only one node
			c.head = nil
			c.tail = nil
		}
	}
}

// get order from cache
func (c *OrderCache) Get(key string) (models.Order, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.store[key]; ok {
		metrics.CacheHits.Inc() // simple stats
		c.moveToFront(node)     // mark as recently used
		return node.value, true
	}

	metrics.CacheMisses.Inc()
	return models.Order{}, false
}
