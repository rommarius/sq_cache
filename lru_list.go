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

// lruList represents a doubly linked list used in a cache system.
type lruList[K IKey, V IValue] struct {
	root *lruListNode[K, V]

	len int
}

// newLRUList creates and returns a new, empty a doubly linked list. The list uses a sentinel root node, and its length
// is initialized to zero.
func newLRUList[K IKey, V IValue]() *lruList[K, V] {
	ll := &lruList[K, V]{
		root: newLRUListNode[K, V](),
	}

	ll.root.next = ll.root
	ll.root.prev = ll.root
	ll.len = 0

	return ll
}

// insert inserts a new node (lln) into the list at the position after the specified node (at).
// The node is properly linked in the doubly linked list.
func (ll *lruList[K, V]) insert(lln, at *lruListNode[K, V]) *lruListNode[K, V] {
	lln.prev = at
	lln.next = at.next
	lln.prev.next = lln
	lln.next.prev = lln
	lln.list = ll
	ll.len++

	return lln
}

// remove removes the specified node (lln) from the list. The node is unlinked and its memory is cleared.
func (ll *lruList[K, V]) remove(lln *lruListNode[K, V]) {
	lln.prev.next = lln.next
	lln.next.prev = lln.prev
	lln.next = nil
	lln.prev = nil
	lln.list = nil
	ll.len--
}

// move moves an existing node (lln) to the position after the specified node (at).
// The list is restructured accordingly.
func (ll *lruList[K, V]) move(lln, at *lruListNode[K, V]) {
	if lln == at {
		return
	}

	lln.prev.next = lln.next
	lln.next.prev = lln.prev

	lln.prev = at
	lln.next = at.next
	lln.prev.next = lln
	lln.next.prev = lln
}

// Len returns the number of nodes currently in the list.
func (ll *lruList[K, V]) Len() int {
	return ll.len
}

// Front returns the first node in the list, or nil if the list is empty.
func (ll *lruList[K, V]) Front() *lruListNode[K, V] {
	if ll.len == 0 {
		return nil
	}

	return ll.root.next
}

// Back returns the last node in the list, or nil if the list is empty.
func (ll *lruList[K, V]) Back() *lruListNode[K, V] {
	if ll.len == 0 {
		return nil
	}

	return ll.root.prev
}

// Remove removes the specified node (lln) from the list and returns it.
func (ll *lruList[K, V]) Remove(lln *lruListNode[K, V]) any {
	if lln.list == ll {
		ll.remove(lln)
	}

	return lln
}

// PushFront inserts a new node (lln) at the front of the list, just after the root node.
func (ll *lruList[K, V]) PushFront(lln *lruListNode[K, V]) *lruListNode[K, V] {
	return ll.insert(lln, ll.root)
}

// PushBack inserts a new node (lln) at the back of the list, just before the root node.
func (ll *lruList[K, V]) PushBack(lln *lruListNode[K, V]) *lruListNode[K, V] {
	return ll.insert(lln, ll.root.prev)
}

// InsertBefore inserts a new node (lln) just before the given node (mark) in the list.
func (ll *lruList[K, V]) InsertBefore(lln, mark *lruListNode[K, V]) *lruListNode[K, V] {
	if mark.list != ll {
		return nil
	}

	return ll.insert(lln, mark.prev)
}

// InsertAfter inserts a new node (lln) just after the given node (mark) in the list.
func (ll *lruList[K, V]) InsertAfter(lln, mark *lruListNode[K, V]) *lruListNode[K, V] {
	if mark.list != ll {
		return nil
	}

	return ll.insert(lln, mark)
}

// MoveToFront moves the specified node (lln) to the front of the list, right after the root node.
func (ll *lruList[K, V]) MoveToFront(lln *lruListNode[K, V]) {
	if lln.list != ll || ll.root.next == lln {
		return
	}
	ll.move(lln, ll.root)
}

// MoveToBack moves the specified node (lln) to the back of the list, right before the root node.
func (ll *lruList[K, V]) MoveToBack(lln *lruListNode[K, V]) {
	if lln.list != ll || ll.root.prev == lln {
		return
	}
	ll.move(lln, ll.root.prev)
}

// MoveBefore moves the specified node (lln) just before another node (mark) in the list.
func (ll *lruList[K, V]) MoveBefore(lln, mark *lruListNode[K, V]) {
	if lln.list != ll || lln == mark || mark.list != ll {
		return
	}
	ll.move(lln, mark.prev)
}

// MoveAfter moves the specified node (lln) just after another node (mark) in the list.
func (ll *lruList[K, V]) MoveAfter(lln, mark *lruListNode[K, V]) {
	if lln.list != ll || lln == mark || mark.list != ll {
		return
	}
	ll.move(lln, mark)
}

// PushBackList appends all nodes from another list (other) to the back of the current list.
func (ll *lruList[K, V]) PushBackList(other *lruList[K, V]) {
	for i, lln := other.Len(), other.Front(); i > 0; i, lln = i-1, lln.Next() {
		ll.insert(lln, ll.root.prev)
	}
}

// PushFrontList appends all nodes from another list (other) to the front of the current list.
func (ll *lruList[K, V]) PushFrontList(other *lruList[K, V]) {
	for i, lln := other.Len(), other.Back(); i > 0; i, lln = i-1, lln.Prev() {
		ll.insert(lln, ll.root)
	}
}
