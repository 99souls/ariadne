package ratelimit

import "time"

type windowBucket struct {
	total  int
	errors int
}

type slidingWindow struct {
	window     time.Duration
	bucketSize time.Duration
	buckets    map[int64]*windowBucket
}

func newSlidingWindow(window, bucket time.Duration) *slidingWindow {
	if bucket <= 0 {
		bucket = time.Second
	}
	if window < bucket {
		window = bucket
	}
	return &slidingWindow{
		window:     window,
		bucketSize: bucket,
		buckets:    make(map[int64]*windowBucket),
	}
}

func (sw *slidingWindow) record(now time.Time, total, errors int) {
	if total == 0 && errors == 0 {
		return
	}
	bucketStart := now.Truncate(sw.bucketSize)
	key := bucketStart.UnixNano()

	if bucket, ok := sw.buckets[key]; ok {
		bucket.total += total
		bucket.errors += errors
	} else {
		sw.buckets[key] = &windowBucket{total: total, errors: errors}
	}

	sw.evict(now)
}

func (sw *slidingWindow) snapshot(now time.Time) (int, int) {
	sw.evict(now)

	total := 0
	errors := 0
	cutoff := now.Add(-sw.window)

	for key, bucket := range sw.buckets {
		start := time.Unix(0, key)
		if start.Before(cutoff) {
			continue
		}
		total += bucket.total
		errors += bucket.errors
	}

	return total, errors
}

func (sw *slidingWindow) errorRate(now time.Time) float64 {
	total, errors := sw.snapshot(now)
	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total)
}

func (sw *slidingWindow) evict(now time.Time) {
	cutoff := now.Add(-sw.window)
	for key := range sw.buckets {
		start := time.Unix(0, key)
		if start.Before(cutoff) {
			delete(sw.buckets, key)
		}
	}
}
