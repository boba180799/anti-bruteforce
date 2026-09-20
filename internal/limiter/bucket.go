// Package limiter реализует алгоритм leaky bucket для ограничения
// частоты попыток авторизации по логину, паролю и IP-адресу.
package limiter

import (
	"sync"
	"time"
)

// Bucket — ведро алгоритма leaky bucket для одного ключа
// (логина, пароля или IP-адреса). Потокобезопасен.
//
// Ёмкость задаёт максимальное количество одновременных запросов,
// а скорость утечки — сколько единиц вытекает в секунду.
type Bucket struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	level    float64
	lastLeak time.Time
}

// NewBucket создаёт ведро с заданной ёмкостью и скоростью утечки.
// capacity — максимум единиц, rate — сколько единиц вытекает за секунду.
func NewBucket(capacity, rate float64) *Bucket {
	return &Bucket{
		capacity: capacity,
		rate:     rate,
		lastLeak: time.Now(),
	}
}

// Allow проверяет, можно ли пропустить один запрос.
// Возвращает true, если запрос разрешён, и false — если ведро переполнено.
func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.leak(time.Now())

	if b.level+1 > b.capacity {
		return false
	}

	b.level++

	return true
}

// Reset обнуляет текущий уровень ведра и время последней утечки.
func (b *Bucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.level = 0
	b.lastLeak = time.Now()
}

// Level возвращает текущий уровень ведра после применения утечки.
// Используется в тестах и метриках.
func (b *Bucket) Level() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.leak(time.Now())

	return b.level
}

// Capacity возвращает ёмкость ведра.
func (b *Bucket) Capacity() float64 {
	return b.capacity
}

// leak вычитает из уровня количество единиц, вытекших с прошлого вызова.
// Предполагается, что вызывающий держит блокировку b.mu.
func (b *Bucket) leak(now time.Time) {
	elapsed := now.Sub(b.lastLeak).Seconds()
	if elapsed <= 0 {
		return
	}

	b.level -= elapsed * b.rate
	if b.level < 0 {
		b.level = 0
	}

	b.lastLeak = now
}
