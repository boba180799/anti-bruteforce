package limiter

import (
	"sync"
	"testing"
	"time"
)

// TestBucket_AllowsUpToCapacity проверяет, что ведро пропускает ровно
// capacity запросов, а следующий — отклоняет.
func TestBucket_AllowsUpToCapacity(t *testing.T) {
	t.Parallel()

	b := NewBucket(5, 0)

	for i := range 5 {
		if !b.Allow() {
			t.Fatalf("attempt %d: expected allow", i+1)
		}
	}

	if b.Allow() {
		t.Fatal("expected deny after capacity exceeded")
	}
}

// TestBucket_LeaksOverTime проверяет, что после паузы уровень ведра
// снижается и запросы снова проходят.
func TestBucket_LeaksOverTime(t *testing.T) {
	t.Parallel()

	b := NewBucket(1, 100)

	if !b.Allow() {
		t.Fatal("first attempt should pass")
	}

	if b.Allow() {
		t.Fatal("second attempt should fail")
	}

	// За 50 мс при скорости 100 ед/с утечёт 5 единиц — ведро опустеет.
	time.Sleep(50 * time.Millisecond)

	if !b.Allow() {
		t.Fatal("attempt should pass after leak")
	}
}

// TestBucket_Reset проверяет, что Reset обнуляет уровень ведра.
func TestBucket_Reset(t *testing.T) {
	t.Parallel()

	b := NewBucket(2, 0)

	_ = b.Allow()
	_ = b.Allow()

	if b.Allow() {
		t.Fatal("expected deny before reset")
	}

	b.Reset()

	if !b.Allow() {
		t.Fatal("expected allow after reset")
	}
}

// TestBucket_ConcurrentAllow проверяет потокобезопасность: при
// одновременных вызовах Allow пропускается ровно capacity запросов.
func TestBucket_ConcurrentAllow(t *testing.T) {
	t.Parallel()

	const (
		capacity = 100
		workers  = 200
	)

	b := NewBucket(capacity, 0)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			if b.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowed != capacity {
		t.Fatalf("expected exactly %d allowed, got %d", capacity, allowed)
	}
}
