// Package usecase содержит бизнес-логику сервиса анти-брутфорс,
// независимую от транспорта и конкретных реализаций хранилищ.
package usecase

// Limiter проверяет и управляет лимитами попыток авторизации.
type Limiter interface {
	AllowByLogin(login string) bool
	AllowByPassword(password string) bool
	AllowByIP(ip string) bool

	ResetLogin(login string)
	ResetIP(ip string)
}

// IPRuleStore управляет whitelist/blacklist CIDR-подсетей.
type IPRuleStore interface {
	IsWhitelisted(ip string) bool
	IsBlacklisted(ip string) bool

	AddWhitelist(cidr string) error
	RemoveWhitelist(cidr string) bool
	AddBlacklist(cidr string) error
	RemoveBlacklist(cidr string) bool

	ListWhitelist() []string
	ListBlacklist() []string
}
