// Package memory содержит in-memory реализации хранилищ сервиса.
package memory

import "github.com/boba180799/anti-bruteforce/internal/netutil"

// IPRules — потокобезопасное in-memory хранилище whitelist/blacklist.
// Реализует интерфейс usecase.IPRuleStore.
type IPRules struct {
	whitelist *netutil.List
	blacklist *netutil.List
}

// NewIPRules создаёт пустое хранилище правил.
func NewIPRules() *IPRules {
	return &IPRules{
		whitelist: netutil.NewList(),
		blacklist: netutil.NewList(),
	}
}

// IsWhitelisted сообщает, входит ли IP в whitelist.
func (r *IPRules) IsWhitelisted(ip string) bool { return r.whitelist.Contains(ip) }

// IsBlacklisted сообщает, входит ли IP в blacklist.
func (r *IPRules) IsBlacklisted(ip string) bool { return r.blacklist.Contains(ip) }

// AddWhitelist добавляет CIDR в whitelist.
func (r *IPRules) AddWhitelist(cidr string) error { return r.whitelist.Add(cidr) }

// RemoveWhitelist удаляет CIDR из whitelist.
func (r *IPRules) RemoveWhitelist(cidr string) bool { return r.whitelist.Remove(cidr) }

// AddBlacklist добавляет CIDR в blacklist.
func (r *IPRules) AddBlacklist(cidr string) error { return r.blacklist.Add(cidr) }

// RemoveBlacklist удаляет CIDR из blacklist.
func (r *IPRules) RemoveBlacklist(cidr string) bool { return r.blacklist.Remove(cidr) }

// ListWhitelist возвращает все CIDR из whitelist.
func (r *IPRules) ListWhitelist() []string { return r.whitelist.All() }

// ListBlacklist возвращает все CIDR из blacklist.
func (r *IPRules) ListBlacklist() []string { return r.blacklist.All() }
