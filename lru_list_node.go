// BSD 3-Clause License
//
// Copyright © 2009 - The Go Authors. All rights reserved.
//
// Copyright © 2025 - Marius Romeiser. All rights reserved.
//
// https://pkg.go.dev/container/list
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
	"time"
)

// lruListNode represents a node in a doubly linked list used in a cache system.
type lruListNode[K IKey, V IValue] struct {
	next *lruListNode[K, V]
	prev *lruListNode[K, V]

	list *lruList[K, V]

	Key   K
	Value V
	TTL   time.Time
}

// newLRUListNode creates and returns a new lruListNode instance.
func newLRUListNode[K IKey, V IValue]() *lruListNode[K, V] {
	lln := &lruListNode[K, V]{}
	return lln
}

// Next returns the next node in the list, or nil if there is no next node or if the list is invalid.
func (lln *lruListNode[K, V]) Next() *lruListNode[K, V] {
	if p := lln.next; lln.list != nil && p != lln.list.root {
		return p
	}

	return nil
}

// Prev returns the previous node in the list, or nil if there is no previous node or if the list is invalid.
func (lln *lruListNode[K, V]) Prev() *lruListNode[K, V] {
	if p := lln.prev; lln.list != nil && p != lln.list.root {
		return p
	}

	return nil
}
