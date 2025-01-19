# sq_cache

The "github.com/mariusromeiser/sq_cache" package is a highly efficient, flexible caching library designed to enhance data retrieval and memory management in applications that require fast data access. It provides a robust solution to caching with a focus on optimizing both speed and memory usage. Using the Least Recently Used (LRU) eviction strategy, "github.com/mariusromeiser/sq_cache" ensures that the least recently accessed cache entries are evicted when the cache reaches its capacity, making it an ideal choice for high-performance caching scenarios.

The powerful library is suitable for applications requiring high-performance caching with fine-tuned memory management. Its flexibility, including support for TTL, periodic cleanup, and detailed telemetry, makes it a perfect choice for building efficient caching systems in a wide range of use cases.

## Installation

```bash
go install https://github.com/mariusromeiser/sq_cache@latest
```

## Documentation

[Documentation](https://pkg.go.dev/github.com/mariusromeiser/sq_cache) 

https://pkg.go.dev/github.com/mariusromeiser/sq_cache

## Usage / Examples

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "github.com/mariusromeiser/sq_cache"
)

func main() {
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    ctx, cancel := context.WithCancel(context.Background())

    config := &sq_cache.Config[string, []byte]{
        LoggingOn:   true,
        TelemetryOn: true,

        MaxShards: 256,
        MaxItems:  1000000,

        ExpiryDurationInSeconds:  60 * 60 * 24,
        CleanupDurationInSeconds: 60 * 5,
    }

    cache, err := sq_cache.NewLRUCache(ctx, config)
    if err != nil {
        panic(err)
    }

    // define operational functions

    <-stop
    cancel()
}
```

## Define custom key generation function

```go
generateKey := func[K string, V []byte](value V) (key K) {
    // define custom key generation function
}

config := &sq_cache.Config[string, []byte]{
    GenerateKey: generateKey,
}
```

## Define custom shard id generation function

```go
generateShardId := func[K string, V []byte](key K, maxShards uint) (shardId uint) {
    // define custom shard id generation function 
}

config := &sq_cache.Config[string, []byte]{
    GenerateShardId: generateShardId,
}
```

## Define custom callback functions

### OnAdd

```go
onAdd := func[K string, V []byte](loggingOn bool, node *sq_cache.LRUListNode[K, V]) {
    // define custom callback function
}

config := &sq_cache.Config[string, []byte]{
    OnAdd: onAdd,
}
```

### OnUpdate

```go
onUpdate := func[K string, V []byte](loggingOn bool, node *sq_cache.LRUListNode[K, V]) {
    // define custom callback function
}

config := &sq_cache.Config[string, []byte]{
    OnUpdate: onUpdate,
}
```

### OnHit

```go
onHit := func[K string, V []byte](loggingOn bool, node *sq_cache.LRUListNode[K, V]) {
    // define custom callback function
}

config := &sq_cache.Config[string, []byte]{
    OnHit: onHit,
}
```

### OnMiss

```go
onMiss := func[K string, V []byte](loggingOn bool, key K) {
    // define custom callback function
}

config := &sq_cache.Config[string, []byte]{
    OnMiss: onMiss,
}
```

### OnEvict

```go
onEvict := func[K string, V []byte](loggingOn bool, node *sq_cache.LRUListNode[K, V]) {
    // define custom callback function
}

config := &sq_cache.Config[string, []byte]{
    OnEvict: onEvict,
}
```

## License

BSD 3-Clause License

## Author

Copyright © 2025 - Marius Romeiser. All rights reserved.
