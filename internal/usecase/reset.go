package usecase

// ResetBucket сбрасывает вёдра для логина и IP-адреса.
// Используется администраторами через CLI или API, например
// после того, как легитимный пользователь успешно авторизовался.
type ResetBucket struct {
	limiter Limiter
}

// NewResetBucket создаёт usecase сброса.
func NewResetBucket(limiter Limiter) *ResetBucket {
	return &ResetBucket{limiter: limiter}
}

// Execute сбрасывает вёдра для непустых login и ip.
// Пустые строки игнорируются — это позволяет клиенту сбросить
// только одну из категорий.
func (uc *ResetBucket) Execute(login, ip string) {
	if login != "" {
		uc.limiter.ResetLogin(login)
	}

	if ip != "" {
		uc.limiter.ResetIP(ip)
	}
}
