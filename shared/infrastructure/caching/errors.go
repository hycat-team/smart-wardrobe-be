package caching

import "errors"

// ErrKeyNotFound is returned when a cache key does not exist.
var ErrKeyNotFound = errors.New("cache: key not found")
