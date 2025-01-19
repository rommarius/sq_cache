// BSD 3-Clause License
//
// Copyright © 2025 - Marius Romeiser. All rights reserved.
//
//               \   /                                         \   /
//          .____-/.\-____.      `\\             //'      .____-/.\-____.
//               ~`-'~             \\           //             ~`-'~
//                                  \\. __-__ .//
//                        ___/-_.-.__`/~     ~\'__.-._-\___
// .|.       ___________.'__/__ ~-[ \.\'-----'/./ ]-~ __\__`.___________       .|.
// ~o~~~~~~~--------______-~~~~~-_/_/ |   .   | \_\_-~~~~~-______--------~~~~~~~o~
// ' `               + + +  (X)(X)  ~--\__ __/--~  (X)(X)  + + +               ' `
//                              (X) `/.\' ~ `/.\' (X)
//                                  "\_/"   "\_/"
//                                __
//    _________ ___  ______ _____/ /________  ____
//   / ___/ __ `/ / / / __ `/ __  / ___/ __ \/ __ \
//  (__  ) /_/ / /_/ / /_/ / /_/ / /  / /_/ / / / /
// /____/\__, /\__,_/\__,_/\__,_/_/   \____/_/ /_/
// squadron/_/
//
// The "sq_cache" package is a highly efficient, flexible caching library designed to enhance data retrieval and memory
// management in applications that require fast data access. It provides a robust solution to caching with a focus on
// optimizing both speed and memory usage. Using the Least Recently Used (LRU) eviction strategy, "sq_cache" ensures
// that the least recently accessed cache entries are evicted when the cache reaches its capacity, making it an ideal
// choice for high-performance caching scenarios.
//
// The powerful library is suitable for applications requiring high-performance caching with fine-tuned memory
// management. Its flexibility, including support for TTL, periodic cleanup, and detailed telemetry, makes it a perfect
// choice for building efficient caching systems in a wide range of use cases.

package sq_cache

import (
	"sync"
	"time"

	"github.com/mariusromeiser/generic_syncpool"
)

// lruCacheShard represents a non thread-safe lru (Least Recently Used) cache shard.
type lruCacheShard[K IKey, V IValue] struct {
	sync.RWMutex

	id int64

	maxItems int64

	loggingOn   bool
	telemetryOn bool

	list      *lruList[K, V]
	nodesPool *generic_syncpool.Pool[lruListNode[K, V]]
	nodes     map[K]*lruListNode[K, V]

	telemetry *telemetry

	onAdd    func(loggingOn bool, node *lruListNode[K, V])
	onUpdate func(loggingOn bool, node *lruListNode[K, V])
	onHit    func(loggingOn bool, node *lruListNode[K, V])
	onMiss   func(loggingOn bool, key K)
	onEvict  func(loggingOn bool, node *lruListNode[K, V])
}

// newLRUCacheShard initializes and returns a new lruCacheShard instance with user-configured settings.
func newLRUCacheShard[K IKey, V IValue](config *Config[K, V], id int64) (shard *lruCacheShard[K, V]) {
	shard = &lruCacheShard[K, V]{
		id: id,

		maxItems: config.MaxItems,

		loggingOn:   config.LoggingOn,
		telemetryOn: config.TelemetryOn,

		list:      newLRUList[K, V](),
		nodesPool: generic_syncpool.New[lruListNode[K, V]](),
		nodes:     make(map[K]*lruListNode[K, V], config.MaxItems),

		telemetry: newTelemetry(),

		onAdd:    config.OnAdd,
		onUpdate: config.OnUpdate,
		onHit:    config.OnHit,
		onMiss:   config.OnMiss,
		onEvict:  config.OnEvict,
	}

	return shard
}

// getItemFromPool retrieves an item from the cache pool.
// Used for efficient memory management and allocation to minimize overhead and optimize resource usage in caching
// operations.
func (shard *lruCacheShard[K, V]) getItemFromPool(key K, value V, ttl time.Time) (item *lruListNode[K, V]) {
	item = shard.nodesPool.Get()
	item.Key = key
	item.Value = value
	item.TTL = ttl

	return item
}

// putItemInPool puts an item back into the pool.
// Used for efficient memory management and allocation to minimize overhead and optimize resource usage in caching
// operations.
func (shard *lruCacheShard[K, V]) putItemInPool(item *lruListNode[K, V]) {
	item.Key = ""
	item.Value = nil
	item.TTL = time.Time{}
	shard.nodesPool.Put(item)
}

// removeItemOldest removes the oldest (least recently used) item from the shard.
func (shard *lruCacheShard[K, V]) removeItemOldest() {
	if item := shard.list.Back(); item != nil {
		shard.removeItem(item)
	}
}

// removeItem removes a specific item from the shard by reference.
func (shard *lruCacheShard[K, V]) removeItem(item *lruListNode[K, V]) {
	shard.list.Remove(item)
	delete(shard.nodes, item.Key)

	if shard.telemetryOn {
		shard.onEvict(shard.loggingOn, item)
	}
}

// CleanupShard handles the periodic cleanup of the shard.
func (shard *lruCacheShard[K, V]) CleanupShard() (evictCount int64) {
	now := time.Now()
	for _, item := range shard.nodes {
		if item.TTL.Before(now) && !item.TTL.IsZero() {
			shard.removeItem(item)
			evictCount++
		}
	}

	return evictCount
}

// Set adds a key-value pair with a specific TTL (time to live) to the shard.
// This operation does updates the recent-ness of the cache item.
func (shard *lruCacheShard[K, V]) Set(cacheLen int64, key K, value V, ttl time.Time) (evicted, added bool) {
	newItem := shard.getItemFromPool(key, value, ttl)

	if item, found := shard.nodes[key]; found {
		shard.list.MoveToFront(item)
		item.Value = value
		item.TTL = ttl

		if shard.telemetryOn {
			shard.onUpdate(shard.loggingOn, item)
		}

		return false, false
	} else {
		item := shard.list.PushFront(newItem)
		shard.nodes[key] = item

		if shard.telemetryOn {
			shard.onAdd(shard.loggingOn, item)
		}

		evict := cacheLen >= shard.maxItems
		if evict {
			shard.removeItemOldest()
		    return true, false
		}

		return false, true
	}
}

// Get retrieves a value by the specified key from the shard.
// This operation does updates the recent-ness of the cache item.
func (shard *lruCacheShard[K, V]) Get(key K) (value V, found bool) {
	if item, found := shard.nodes[key]; found {
		shard.list.MoveToFront(item)

		if shard.telemetryOn {
			shard.onHit(shard.loggingOn, item)
		}

		return item.Value, true
	} else {
		if shard.telemetryOn {
			shard.onMiss(shard.loggingOn, key)
		}

		return *new(V), false
	}
}

// Contains checks if a specified key exists in the shard.
// This operation doesn't updates the recent-ness of the cache item.
func (shard *lruCacheShard[K, V]) Contains(key K) (found bool) {
	if item, found := shard.nodes[key]; found {
		if shard.telemetryOn {
			shard.onHit(shard.loggingOn, item)
		}

		return true
	} else {
		if shard.telemetryOn {
			shard.onMiss(shard.loggingOn, key)
		}

		return false
	}
}

// Peek retrieves a value by the specified key from the shard.
// This operation doesn't updates the recent-ness of the cache item.
func (shard *lruCacheShard[K, V]) Peek(key K) (value V, found bool) {
	if item, found := shard.nodes[key]; found {
		if shard.telemetryOn {
			shard.onHit(shard.loggingOn, item)
		}

		return item.Value, true
	} else {
		if shard.telemetryOn {
			shard.onMiss(shard.loggingOn, key)
		}

		return *new(V), false
	}
}

// Remove removes a key-value pair from the shard.
func (shard *lruCacheShard[K, V]) Remove(key K) (removed bool) {
	if item, found := shard.nodes[key]; found {
		shard.removeItem(item)

		return true
	} else {
		if shard.telemetryOn {
			shard.onMiss(shard.loggingOn, key)
		}

		return false
	}
}

// Purge clears all items in the shard.
func (shard *lruCacheShard[K, V]) Purge() {
	shard.list = newLRUList[K, V]()
	shard.nodesPool = generic_syncpool.New[lruListNode[K, V]]()
	shard.nodes = make(map[K]*lruListNode[K, V], shard.maxItems)
}

// telemetry returns the shard's telemetry (add, update, hit, miss, evict counters).
func (shard *lruCacheShard[K, V]) Telemetry() (telemetry *telemetry) {
	return shard.telemetry
}

// telemetryReset resets the shard's telemetry counters (add, update, hit, miss, evict) to zero.
func (shard *lruCacheShard[K, V]) TelemetryReset() {
	shard.telemetry = newTelemetry()
}
