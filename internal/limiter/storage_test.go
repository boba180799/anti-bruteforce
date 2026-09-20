package limiter

import (
	"context"
	"testing"
	"time"
)

// TestStorage_IndependentBuckets проверяет, что разные логины
// имеют независимые вёдра.
func TestStorage_IndependentBuckets(t *testing.T) {
	t.Parallel()

	s := NewStorage(StorageConfig{
		LoginCapacity: 2,
		LoginRate:     0,
		TTL:           time.Minute,
	})

	if !s.AllowByLogin("alice") {
		t.Fatal("alice 1 should pass")
	}
	if !s.AllowByLogin("alice") {
		t.Fatal("alice 2 should pass")
	}
	if s.AllowByLogin("alice") {
		t.Fatal("alice 3 should be denied")
	}

	if !s.AllowByLogin("bob") {
		t.Fatal("bob should have its own bucket")
	}
}

// TestStorage_ThreeDimensions проверяет, что login, password и ip
// используют разные наборы вёдер и не пересекаются.
func TestStorage_ThreeDimensions(t *testing.T) {
	t.Parallel()

	s := NewStorage(StorageConfig{
		LoginCapacity:    1,
		LoginRate:        0,
		PasswordCapacity: 1,
		PasswordRate:     0,
		IPCapacity:       1,
		IPRate:           0,
		TTL:              time.Minute,
	})

	if !s.AllowByLogin("u") {
		t.Fatal("login bucket should allow once")
	}
	if s.AllowByLogin("u") {
		t.Fatal("login bucket should be full")
	}

	// Тот же ключ "u", но в другой категории — отдельное ведро.
	if !s.AllowByPassword("u") {
		t.Fatal("password bucket is independent")
	}
	if !s.AllowByIP("u") {
		t.Fatal("ip bucket is independent")
	}

	if s.Len() != 3 {
		t.Fatalf("expected 3 buckets, got %d", s.Len())
	}
}

// TestStorage_CleanupRemovesInactive проверяет, что неактивные вёдра
// удаляются фоновой очисткой после истечения TTL.
func TestStorage_CleanupRemovesInactive(t *testing.T) {
	t.Parallel()

	ttl := 100 * time.Millisecond
	s := NewStorage(StorageConfig{
		LoginCapacity: 1,
		LoginRate:     0,
		TTL:           ttl,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Start(ctx)

	_ = s.AllowByLogin("ghost")

	if s.Len() != 1 {
		t.Fatalf("expected 1 bucket, got %d", s.Len())
	}

	time.Sleep(3 * ttl)

	if s.Len() != 0 {
		t.Fatalf("expected bucket to be cleaned, got %d", s.Len())
	}
}

// TestStorage_CleanupStopsOnContext проверяет, что после отмены
// контекста фоновая очистка завершается.
func TestStorage_CleanupStopsOnContext(t *testing.T) {
	t.Parallel()

	s := NewStorage(StorageConfig{
		LoginCapacity: 1,
		TTL:           20 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	_ = s.AllowByLogin("x")

	cancel()

	// Через несколько TTL вёдра не должны очищаться, потому что
	// горутина уже вышла. Проверяем косвенно — количество не изменилось.
	time.Sleep(100 * time.Millisecond)

	if s.Len() != 1 {
		t.Fatalf("cleanup should be stopped, but bucket count = %d", s.Len())
	}
}
