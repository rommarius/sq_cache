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
	"crypto/sha1"
	"encoding/binary"
	"log"
)

// generateKey generates a hash key from a specified value.
func generateKey[K IKey, V IValue](value V) (key K) {
	h := sha1.New()

	switch v := any(value).(type) {
	case []byte:
		h.Write(v)
		k := h.Sum(nil)
		h.Reset()
		return any(k).(K)
	default:
		panic("IValue type is not supported")
	}
}

// generateShardId generates a shardId based on the specified key and the maximum number of shards.
func generateShardId[K IKey](key K, maxItems int64) (shardId int64) {
	h := sha1.New()

	switch k := any(key).(type) {
	case string:
		h.Write([]byte(k))
		v := h.Sum(nil)
		h.Reset()
		return int64(binary.BigEndian.Uint64(v) % uint64(maxItems))
	default:
		panic("IKey type is not supported")
	}
}

// onAdd is a callback function that gets triggered when a cache item is added.
func onAdd[K IKey, V IValue](loggingOn bool, node *lruListNode[K, V]) {
	if loggingOn {
		log.Printf(
			"%s: onAdd callback - key - %+v", LibraryName, node.Key,
		)
	}
}

// onUpdate is a callback function that gets triggered when a cache item gets updated.
func onUpdate[K IKey, V IValue](loggingOn bool, node *lruListNode[K, V]) {
	if loggingOn {
		log.Printf(
			"%s: onUpdate callback - key - %+v", LibraryName, node.Key,
		)
	}
}

// onHit is a callback function that gets triggered when a cache item is found.
func onHit[K IKey, V IValue](loggingOn bool, node *lruListNode[K, V]) {
	if loggingOn {
		log.Printf(
			"%s: onHit callback - key - %+v", LibraryName, node.Key,
		)
	}
}

// onMiss is a callback function that gets triggered when a cache item is not found.
func onMiss[K IKey, V IValue](loggingOn bool, key K) {
	if loggingOn {
		log.Printf(
			"%s: onMiss callback - key: %+v", LibraryName, key,
		)
	}

}

// onEvict is a callback function that gets triggered when a cache item gets evcited.
func onEvict[K IKey, V IValue](loggingOn bool, node *lruListNode[K, V]) {
	if loggingOn {
		log.Printf(
			"%s: onEvict callback - key - %+v", LibraryName, node.Key,
		)
	}
}
