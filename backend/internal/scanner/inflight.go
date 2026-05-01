package scanner

import "sync/atomic"

// inFlight is true while a scan (manual or auto) is running. Both paths
// share this flag so the UI's /api/scan/status reflects activity from
// either trigger source.
var inFlight int32

// IsRunning reports whether a scan is currently in flight.
func IsRunning() bool { return atomic.LoadInt32(&inFlight) == 1 }

// TryAcquire claims the scan lock if no scan is running. Returns false
// if another scan is already in flight.
func TryAcquire() bool { return atomic.CompareAndSwapInt32(&inFlight, 0, 1) }

// Release frees the scan lock.
func Release() { atomic.StoreInt32(&inFlight, 0) }
