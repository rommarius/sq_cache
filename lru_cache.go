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
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rommarius/sq_config_combine"
)

// CacheStatus defines the status of the cache.
type CacheStatus int

// CacheStatus modes
const (
	Opened CacheStatus = iota
	Started
	Stopped
	Closed
)

// LRUCache represents an a thread-safe LRU (Least Recently Used) cache.
type LRUCache[K IKey, V IValue] struct {
	ctx context.Context

	maxShards int64
	maxItems  int64
	len       atomic.Int64

	loggingOn   bool
	telemetryOn bool

	expiryDurationInSeconds  int64
	cleanupDurationInSeconds int64

	generateKey     func(value V) K
	generateShardId func(key K, maxItems int64) int64

	status                CacheStatus
	isCleanupActive       chan bool
	isCleanupTickerActive bool

	shards []*lruCacheShard[K, V]
}

// NewLRUCache initializes and returns a new LRUCache instance with user-configured settings.
//
// Parameters:
//   - ctx: The context to manage the lifecycle of the cache.
//   - userConfig: A user-defined configuration for customizing the cache's behavior.
//
// Returns:
//   - cache: The created LRUCache object.
//   - err: An error, if any occurs during initialization.
func NewLRUCache[K IKey, V IValue](ctx context.Context, userConfig *Config[K, V]) (cache *LRUCache[K, V], err error) {
	defaultConfig := &Config[K, V]{
		MaxShards: 256,
		MaxItems:  1000000,

		LoggingOn:   true,
		TelemetryOn: true,

		ExpiryDurationInSeconds:  60 * 60 * 24,
		CleanupDurationInSeconds: 60 * 5,

		GenerateKey:     generateKey[K, V],
		GenerateShardId: generateShardId[K],

		OnAdd:    onAdd[K, V],
		OnUpdate: onUpdate[K, V],
		OnHit:    onHit[K, V],
		OnMiss:   onMiss[K, V],
		OnEvict:  onEvict[K, V],
	}

	sq, err := sq_config_combine.New[Config[K, V]](defaultConfig, userConfig)
	config := sq.Combine()
	if err != nil {
		return nil, err
	}

	cache = &LRUCache[K, V]{
		ctx: ctx,

		maxShards: config.MaxShards,
		maxItems:  config.MaxItems,

		loggingOn: config.LoggingOn,

		expiryDurationInSeconds:  config.ExpiryDurationInSeconds,
		cleanupDurationInSeconds: config.CleanupDurationInSeconds,

		generateKey:     config.GenerateKey,
		generateShardId: config.GenerateShardId,

		status:                Opened,
		isCleanupTickerActive: true,
		isCleanupActive:       make(chan bool),

		shards: make([]*lruCacheShard[K, V], config.MaxShards),
	}

	for shardId := range cache.shards {
		cache.shards[shardId] = newLRUCacheShard[K, V](config, int64(shardId))
	}

	cache.Start()

	go cache.cleanupTicker()

	return cache, nil
}

// cleanupStart activates the cache cleanup process.
func (cache *LRUCache[K, V]) cleanupStart() {
	cache.isCleanupActive <- true
}

// cleanupStop deactivates the cache cleanup process.
func (cache *LRUCache[K, V]) cleanupStop() {
	cache.isCleanupActive <- false
}

// cleanupTicker handles the periodic cleanup of the cache.
func (cache *LRUCache[K, V]) cleanupTicker() {
	ticker := time.NewTicker(time.Second * time.Duration(cache.cleanupDurationInSeconds))

	if cache.loggingOn {
		log.Println("cache started the cleanup process.")
	}

	for {
		select {
		case active := <-cache.isCleanupActive:
			cache.isCleanupTickerActive = active
		case <-ticker.C:
			if cache.isCleanupTickerActive {
				cache.cleanupShards()
				if cache.loggingOn {
					log.Println("cache shards are cleaned up.")
				}
			}
		case <-cache.ctx.Done():
			ticker.Stop()
			if cache.loggingOn {
				log.Println("cache stopped the cleanup process.")
			}
			return
		}
	}
}

// cleanupShards handles the periodic cleanup of the cache shards.
func (cache *LRUCache[K, V]) cleanupShards() {
	var wg sync.WaitGroup

	for shardId := range cache.shards {
		wg.Add(1)

		go func(shardId int) {
			defer wg.Done()

			cache.shards[shardId].Lock()
			evictCount := cache.shards[shardId].CleanupShard()
			cache.shards[shardId].Unlock()

			cache.len.Add(evictCount)
		}(shardId)
	}

	wg.Wait()
}

// Status returns the current status of the cache (Opened, Started, Stopped, Closed).
//
// Returns:
//   - status: The current status of the cache.
func (cache *LRUCache[K, V]) Status() (status CacheStatus) {
	return cache.status
}

// Start activates the cache, allowing operations to proceed.
func (cache *LRUCache[K, V]) Start() {
	cache.status = Started
	if cache.loggingOn {
		log.Println("cache is started.")
	}
}

// Stop deactivates the cache, disallowing operations to proceed.
func (cache *LRUCache[K, V]) Stop() {
	cache.status = Stopped
	if cache.loggingOn {
		log.Println("cache is stopped.")
	}
}

// MaxShards returns the configured maximum number of shards in the cache.
//
// Returns:
//   - shards: The configured maximum number of shards in the cache.
func (cache *LRUCache[K, V]) MaxShards() (shards int64) {
	return cache.maxShards
}

// MaxItems returns the configured maximum number of items that can be stored in the cache.
//
// Returns:
//   - items: The configured maximum number of items that the cache can hold.
func (cache *LRUCache[K, V]) MaxItems() (items int64) {
	return cache.maxItems
}

// Len returns the current number of cache items in the cache.
//
// Returns:
//   - len: The current number of cache items in the cache.
func (cache *LRUCache[K, V]) Len() (len int64) {
	return cache.len.Load()
}

// Set adds a key-value pair to the cache.
// If the key wasn't specified, it is generated automatically based on the specified value.
// This operation does updates the recent-ness of the cache item.
//
// Parameters:
//   - key: The key to associate with the value.
//   - value: The value to store in the cache.
//
// Returns:
//   - returnKey: The key that was used for the cache item.
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	_, err := cache.Set("my-key", []byte("my-value"))
//	if err != nil {
//	    panic(err)
//	}
//
//	key, err := cache.Set("", []byte("my-value"))
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) Set(key K, value V) (returnKey K, err error) {
	var k K

	switch cache.Status() {
	case Closed:
		return k, errors.New("cache is closed")
	case Stopped:
		return k, errors.New("cache is stopped, must be started before calling method Set()")
	}

	if key == "" {
		key = cache.generateKey(value)
	}

	var ttl time.Time

	shardId := cache.generateShardId(key, cache.maxItems)

	cache.shards[shardId].Lock()
	evicted, _ := cache.shards[shardId].Set(cache.len.Load(), key, value, ttl)
	cache.shards[shardId].Unlock()
    if evicted {
        cache.len.Add(-1)
    }
	cache.len.Add(1)

	return key, nil
}

// SetWithTTL adds a key-value pair to the cache with a specific TTL (time to live).
// If the key wasn't specified, it is generated automatically based on the specified value.
// If the duration wasn't specified, it uses the default duration time.
// This operation does updates the recent-ness of the cache item.
//
// Parameters:
//   - key: The key to associate with the value.
//   - value: The value to store in the cache.
//   - duration: The time-to-live (TTL) for the cache entry in seconds.
//
// Returns:
//   - returnKey: The key that was used for the cache item.
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	_, err := cache.SetWithTTL("my-key", []byte("my-value"), 3600)
//	if err != nil {
//	    panic(err)
//	}
//
//	key, err := cache.SetWithTTL("", []byte("my-value"), 3600)
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) SetWithTTL(key K, value V, duration uint) (returnKey K, err error) {
	var k K

	switch cache.Status() {
	case Closed:
		return k, errors.New("cache is closed")
	case Stopped:
		return k, errors.New("cache is stopped, must be started before calling method SetWithTTL()")
	}

	if key == "" {
		key = cache.generateKey(value)
	}

	var ttl time.Time
	now := time.Now()
	if duration > 0 {
		ttl = now.Add(time.Duration(duration) * time.Second)
	} else {
		ttl = now.Add(time.Duration(cache.expiryDurationInSeconds) * time.Second)
	}

	shardId := cache.generateShardId(key, cache.maxItems)

	cache.shards[shardId].Lock()
	evicted, _ := cache.shards[shardId].Set(cache.len.Load(), key, value, ttl)
	cache.shards[shardId].Unlock()
    if evicted {
        cache.len.Add(-1)
    }
	cache.len.Add(1)

	return key, nil
}

// Get retrieves a value by the specified key from the cache.
// This operation does updates the recent-ness of the cache item.
//
// Parameters:
//   - key: The key associated with the value to retrieve.
//
// Returns:
//   - value: The value associated with the key if found.
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	value, err := cache.Get("my-key")
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) Get(key K) (value V, err error) {
	var v V

	switch cache.Status() {
	case Closed:
		return v, errors.New("cache is closed")
	case Stopped:
		return v, errors.New("cache is stopped, must be started before calling method Get()")
	}

	shardId := cache.generateShardId(key, cache.maxItems)

	cache.shards[shardId].Lock()
	defer cache.shards[shardId].Unlock()
	value, _ = cache.shards[shardId].Get(key)

	return value, nil
}

// Contains checks if a specified key exists in the cache.
// This operation doesn't updates the recent-ness of the cache item.
//
// Parameters:
//   - key: The key to check for existence in the cache.
//
// Returns:
//   - found: A boolean indicating whether the key exists in the cache.
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	found, err := cache.Contains("my-key")
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) Contains(key K) (found bool, err error) {
	switch cache.Status() {
	case Closed:
		return false, errors.New("cache is closed")
	case Stopped:
		return false, errors.New("cache is stopped, must be started before calling method Contains()")
	}

	shardId := cache.generateShardId(key, cache.maxItems)

	cache.shards[shardId].RLock()
	defer cache.shards[shardId].RUnlock()
	found = cache.shards[shardId].Contains(key)

	return found, nil
}

// Peek retrieves a value by the specified key from the cache.
// This operation doesn't updates the recent-ness of the cache item.
//
// Parameters:
//   - key: The key associated with the value to retrieve.
//
// Returns:
//   - value: The value associated with the key if found.
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	value, err := cache.Peek("my-key")
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) Peek(key K) (value V, err error) {
	var v V

	switch cache.Status() {
	case Closed:
		return v, errors.New("cache is closed")
	case Stopped:
		return v, errors.New("cache is stopped, must be started before calling method Peek()")
	}

	shardId := cache.generateShardId(key, cache.maxItems)

	cache.shards[shardId].RLock()
	defer cache.shards[shardId].RUnlock()
	value, _ = cache.shards[shardId].Peek(key)

	return value, nil
}

// Remove removes a key-value pair from the cache.
//
// Parameters:
//   - key: The key to remove from the cache.
//
// Returns:
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	removed, err := cache.Remove("my-key")
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) Remove(key K) (removed bool, err error) {
	switch cache.Status() {
	case Closed:
		return removed, errors.New("cache is closed")
	case Stopped:
		return removed, errors.New("cache is stopped, must be started before calling method Remove()")
	}

	shardId := cache.generateShardId(key, cache.maxItems)

	cache.shards[shardId].Lock()
	removed = cache.shards[shardId].Remove(key)
	cache.shards[shardId].Unlock()

	cache.len.Add(-1)

	return removed, nil
}

// Purge clears all items in the cache.
//
// Returns:
//   - err: An error if the cache is stopped or closed, or if any other issue occurs.
//
// Example Usage:
//
//	err := cache.Purge("my-key")
//	if err != nil {
//	    panic(err)
//	}
func (cache *LRUCache[K, V]) Purge() (err error) {
	switch cache.Status() {
	case Closed:
		return errors.New("cache is closed")
	case Started:
		return errors.New("cache is started, must be stopped before calling method Purge()")
	}

	for shardId := range cache.shards {
		cache.shards[shardId].Lock()
		cache.shards[shardId].Purge()
		cache.shards[shardId].Unlock()
	}

	return nil
}

// Telemetry returns the cache's aggregated telemetry (add, update, hit, miss, evict counters).
//
// Returns:
//   - telemetry: A pointer to the aggregated cache telemetry.
//   - err: An error if the cache is closed, or if any other issue occurs.
func (cache *LRUCache[K, V]) Telemetry() (telemetry *telemetry, err error) {
	switch cache.Status() {
	case Closed:
		return nil, errors.New("cache is closed")
	}

	if !cache.telemetryOn {
		return nil, errors.New("cache telemetry is disabled")
	}

	for shardId := range cache.shards {
		cache.shards[shardId].RLock()
		shardTelemetry := cache.shards[shardId].Telemetry()
		cache.shards[shardId].RUnlock()

		telemetry.SetHitCounter(
			shardTelemetry.GetHitCounter(),
		)
		telemetry.SetMissCounter(
			shardTelemetry.GetMissCounter(),
		)
		telemetry.SetAddCounter(
			shardTelemetry.GetAddCounter(),
		)
		telemetry.SetUpdateCounter(
			shardTelemetry.GetUpdateCounter(),
		)
		telemetry.SetEvictCounter(
			shardTelemetry.GetEvictCounter(),
		)
	}

	return telemetry, nil
}

// TelemetryReset resets the cache's telemetry counters (add, update, hit, miss, evict) to zero.
//
// Returns:
//   - err: An error if the cache is closed, or if any other issue occurs.
func (cache *LRUCache[K, V]) TelemetryReset() (err error) {
	switch cache.Status() {
	case Closed:
		return errors.New("cache is closed")
	}

	if !cache.telemetryOn {
		return errors.New("cache telemetry is disabled")
	}

	for shardId := range cache.shards {
		cache.shards[shardId].Lock()
		cache.shards[shardId].TelemetryReset()
		cache.shards[shardId].Unlock()
	}

	return nil
}

// Close closes the cache, releasing any resources.
func (cache *LRUCache[K, V]) Close() {
	cache.Stop()

	cache.shards = nil

	cache.status = Closed
	if cache.loggingOn {
		log.Println("cache is closed.")
	}
}
