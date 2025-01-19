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
	"sync/atomic"
)

// counterMode defines the modes of the telemetry.
type counterMode int

// counterMode constants
const (
	Add counterMode = iota
	Update
	Hit
	Miss
	Evict
)

// telemetry is a structure that holds atomic counters for different telemetry metrics.
type telemetry struct {
	Add    atomic.Int64
	Update atomic.Int64
	Hit    atomic.Int64
	Miss   atomic.Int64
	Evict  atomic.Int64
}

// newTelemetry creates and returns a new instance of telemetry with all counters initialized.
func newTelemetry() *telemetry {
	t := &telemetry{}
	return t
}

// getCounter retrieves the value of the specified counter based on the counterMode.
func (t *telemetry) getCounter(mode counterMode) (value int64) {
	switch mode {
	case Add:
		return t.Add.Load()
	case Update:
		return t.Update.Load()
	case Hit:
		return t.Hit.Load()
	case Miss:
		return t.Miss.Load()
	case Evict:
		return t.Evict.Load()
	default:
		panic("counterMode doesn't exists")
	}
}

// setCounter sets the value of the specified counter based on the counterMode.
func (t *telemetry) setCounter(mode counterMode, value int64) {
	switch mode {
	case Add:
		t.Add.Store(value)
	case Update:
		t.Update.Store(value)
	case Hit:
		t.Hit.Store(value)
	case Miss:
		t.Miss.Store(value)
	case Evict:
		t.Evict.Store(value)
	default:
		panic("counterMode doesn't exists")
	}
}

// GetAddCounter retrieves the current value of the "Add" counter.
func (t *telemetry) GetAddCounter() (value int64) {
	return t.getCounter(Add)
}

// SetAddCounter Sets the value of the "Add" counter.
func (t *telemetry) SetAddCounter(value int64) {
	t.setCounter(Add, value)
}

// GetUpdateCounter retrieves the current value of the "Update" counter.
func (t *telemetry) GetUpdateCounter() (value int64) {
	return t.getCounter(Update)
}

// SetUpdateCounter Sets the value of the "Update" counter.
func (t *telemetry) SetUpdateCounter(value int64) {
	t.setCounter(Update, value)
}

// GetHitCounter retrieves the current value of the "Hit" counter.
func (t *telemetry) GetHitCounter() (value int64) {
	return t.getCounter(Hit)
}

// SetHitCounter Sets the value of the "Hit" counter.
func (t *telemetry) SetHitCounter(value int64) {
	t.setCounter(Hit, value)
}

// GetMissCounter retrieves the current value of the "Miss" counter.
func (t *telemetry) GetMissCounter() (value int64) {
	return t.getCounter(Miss)
}

// SetMissCounter Sets the value of the "Miss" counter.
func (t *telemetry) SetMissCounter(value int64) {
	t.setCounter(Miss, value)
}

// GetEvictCounter retrieves the current value of the "Evict" counter.
func (t *telemetry) GetEvictCounter() (value int64) {
	return t.getCounter(Evict)
}

// SetEvictCounter Sets the value of the "Evict" counter.
func (t *telemetry) SetEvictCounter(value int64) {
	t.setCounter(Evict, value)
}
