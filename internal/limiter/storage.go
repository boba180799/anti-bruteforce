package limiter

import (
	"context"
	"sync"
	"time"
)

// StorageConfig задаёт лимиты для каждого типа вёдер и TTL неактивных.
type StorageConfig struct {
	LoginCapacity    float64
	LoginRate        float64
	PasswordCapacity float64
	PasswordRate     float64
	IPCapacity       float64
	IPRate           float64
	// TTL определяет, как долго неактивное ведро остаётся в памяти.
	TTL time.Duration
}

// Storage хранит вёдра по ключам и автоматически удаляет неактивные.
// Безопасен для конкурентного использования.
type Storage struct {
	mu      sync.Mutex
	buckets map[string]*entry
	cfg     StorageConfig
}

// entry — обёртка над ведром, хранящая время последнего обращения.
type entry struct {
	bucket       *Bucket
	lastActivity time.Time
}

// NewStorage создаёт пустое хранилище с заданной конфигурацией.
func NewStorage(cfg StorageConfig) *Storage {
	return &Storage{
		buckets: make(map[string]*entry),
		cfg:     cfg,
	}
}

// Start запускает фоновую очистку неактивных вёдер.
// Возвращает управление немедленно; очистка остановится при отмене ctx.
func (s *Storage) Start(ctx context.Context) {
	interval := s.cfg.TTL / 2
	if interval <= 0 {
		interval = time.Minute
	}

	go s.cleanupLoop(ctx, interval)
}

// AllowByLogin проверяет лимит попыток для указанного логина.
func (s *Storage) AllowByLogin(login string) bool {
	return s.allow("login:"+login, s.cfg.LoginCapacity, s.cfg.LoginRate)
}

// AllowByPassword проверяет лимит попыток для указанного пароля.
func (s *Storage) AllowByPassword(password string) bool {
	return s.allow("password:"+password, s.cfg.PasswordCapacity, s.cfg.PasswordRate)
}

// AllowByIP проверяет лимит попыток для указанного IP.
func (s *Storage) AllowByIP(ip string) bool {
	return s.allow("ip:"+ip, s.cfg.IPCapacity, s.cfg.IPRate)
}

// Len возвращает текущее количество вёдер в хранилище.
// Используется в тестах и метриках.
func (s *Storage) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.buckets)
}

// allow — общая логика для всех трёх типов: получить или создать ведро,
// обновить время активности и проверить Allow.
func (s *Storage) allow(key string, capacity, rate float64) bool {
	s.mu.Lock()

	e, ok := s.buckets[key]
	if !ok {
		e = &entry{bucket: NewBucket(capacity, rate)}
		s.buckets[key] = e
	}
	e.lastActivity = time.Now()

	s.mu.Unlock()

	return e.bucket.Allow()
}

// cleanupLoop периодически вызывает cleanup до отмены ctx.
func (s *Storage) cleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup удаляет вёдра, к которым не обращались дольше TTL.
func (s *Storage) cleanup() {
	threshold := time.Now().Add(-s.cfg.TTL)

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, e := range s.buckets {
		if e.lastActivity.Before(threshold) {
			delete(s.buckets, k)
		}
	}
}
