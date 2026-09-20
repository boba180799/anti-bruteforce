// Package netutil содержит утилиты для работы с IP-адресами
// и CIDR-подсетями, используемые при проверке whitelist/blacklist.
package netutil

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// ErrInvalidCIDR возвращается при попытке добавить некорректную подсеть.
var ErrInvalidCIDR = errors.New("invalid CIDR")

// List — потокобезопасный набор CIDR-подсетей.
// Используется для хранения whitelist и blacklist.
type List struct {
	mu   sync.RWMutex
	nets map[string]*net.IPNet
}

// NewList создаёт пустой список подсетей.
func NewList() *List {
	return &List{
		nets: make(map[string]*net.IPNet),
	}
}

// Add добавляет подсеть в формате CIDR (например, "192.168.1.0/24").
// Если подсеть уже есть, повторное добавление не считается ошибкой.
// Возвращает ErrInvalidCIDR, если строка не является валидным CIDR.
func (l *List) Add(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidCIDR, cidr)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.nets[ipnet.String()] = ipnet

	return nil
}

// Remove удаляет подсеть из списка.
// Возвращает true, если подсеть была найдена и удалена.
func (l *List) Remove(cidr string) bool {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	key := ipnet.String()
	if _, ok := l.nets[key]; !ok {
		return false
	}

	delete(l.nets, key)

	return true
}

// Contains проверяет, входит ли указанный IP в одну из подсетей списка.
// Возвращает false для невалидного IP или пустого списка.
func (l *List) Contains(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, ipnet := range l.nets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

// All возвращает срез всех подсетей в строковом формате.
// Порядок не гарантирован.
func (l *List) All() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]string, 0, len(l.nets))
	for k := range l.nets {
		out = append(out, k)
	}

	return out
}

// Len возвращает количество подсетей в списке.
func (l *List) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.nets)
}
