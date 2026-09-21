// Package memory содержит in-memory реализации хранилищ сервиса.
package memory

import (
	"net"
	"sync"

	"github.com/boba180799/anti-bruteforce/internal/netutil"
	"github.com/boba180799/anti-bruteforce/internal/usecase"
)

// IPRules — потокобезопасное in-memory хранилище whitelist/blacklist
// с описаниями правил. Реализует usecase.IPRuleStore.
type IPRules struct {
	mu          sync.RWMutex
	whitelist   *netutil.List
	blacklist   *netutil.List
	whitelistMD map[string]string // CIDR → description
	blacklistMD map[string]string
}

// NewIPRules создаёт пустое хранилище правил.
func NewIPRules() *IPRules {
	return &IPRules{
		whitelist:   netutil.NewList(),
		blacklist:   netutil.NewList(),
		whitelistMD: make(map[string]string),
		blacklistMD: make(map[string]string),
	}
}

// IsWhitelisted сообщает, входит ли IP в whitelist.
func (r *IPRules) IsWhitelisted(ip string) bool { return r.whitelist.Contains(ip) }

// IsBlacklisted сообщает, входит ли IP в blacklist.
func (r *IPRules) IsBlacklisted(ip string) bool { return r.blacklist.Contains(ip) }

// AddWhitelist добавляет CIDR в whitelist с описанием.
func (r *IPRules) AddWhitelist(cidr, description string) error {
	normalized, err := normalizeCIDR(cidr)
	if err != nil {
		return err
	}

	if err := r.whitelist.Add(normalized); err != nil {
		return err
	}

	r.mu.Lock()
	r.whitelistMD[normalized] = description
	r.mu.Unlock()

	return nil
}

// RemoveWhitelist удаляет CIDR из whitelist.
func (r *IPRules) RemoveWhitelist(cidr string) bool {
	normalized, err := normalizeCIDR(cidr)
	if err != nil {
		return false
	}

	if !r.whitelist.Remove(normalized) {
		return false
	}

	r.mu.Lock()
	delete(r.whitelistMD, normalized)
	r.mu.Unlock()

	return true
}

// AddBlacklist добавляет CIDR в blacklist с описанием.
func (r *IPRules) AddBlacklist(cidr, description string) error {
	normalized, err := normalizeCIDR(cidr)
	if err != nil {
		return err
	}

	if err := r.blacklist.Add(normalized); err != nil {
		return err
	}

	r.mu.Lock()
	r.blacklistMD[normalized] = description
	r.mu.Unlock()

	return nil
}

// RemoveBlacklist удаляет CIDR из blacklist.
func (r *IPRules) RemoveBlacklist(cidr string) bool {
	normalized, err := normalizeCIDR(cidr)
	if err != nil {
		return false
	}

	if !r.blacklist.Remove(normalized) {
		return false
	}

	r.mu.Lock()
	delete(r.blacklistMD, normalized)
	r.mu.Unlock()

	return true
}

// ListWhitelist возвращает все правила whitelist.
func (r *IPRules) ListWhitelist() []usecase.RuleInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]usecase.RuleInfo, 0, len(r.whitelistMD))
	for cidr, desc := range r.whitelistMD {
		out = append(out, usecase.RuleInfo{CIDR: cidr, Description: desc})
	}

	return out
}

// ListBlacklist возвращает все правила blacklist.
func (r *IPRules) ListBlacklist() []usecase.RuleInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]usecase.RuleInfo, 0, len(r.blacklistMD))
	for cidr, desc := range r.blacklistMD {
		out = append(out, usecase.RuleInfo{CIDR: cidr, Description: desc})
	}

	return out
}

// normalizeCIDR приводит CIDR к канонической форме ("192.168.1.5/24" → "192.168.1.0/24").
func normalizeCIDR(cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}

	return ipnet.String(), nil
}
