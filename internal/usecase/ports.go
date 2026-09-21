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

// RuleType определяет тип CIDR-правила.
type RuleType int

const (
	// RuleTypeUnspecified — правило без типа (используется как фильтр «все»).
	RuleTypeUnspecified RuleType = iota

	// RuleTypeWhitelist — доверенные подсети.
	RuleTypeWhitelist

	// RuleTypeBlacklist — запрещённые подсети.
	RuleTypeBlacklist
)

// RuleInfo — информация о CIDR-правиле для отображения клиентам.
type RuleInfo struct {
	CIDR        string
	Description string
}

// IPRuleStore управляет whitelist/blacklist CIDR-подсетей.
type IPRuleStore interface {
	IsWhitelisted(ip string) bool
	IsBlacklisted(ip string) bool

	AddWhitelist(cidr, description string) error
	RemoveWhitelist(cidr string) bool
	AddBlacklist(cidr, description string) error
	RemoveBlacklist(cidr string) bool

	ListWhitelist() []RuleInfo
	ListBlacklist() []RuleInfo
}
