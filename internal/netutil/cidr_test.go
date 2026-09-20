package netutil

import (
	"errors"
	"sort"
	"testing"
)

// TestList_AddAndContains проверяет базовое добавление и проверку вхождения.
func TestList_AddAndContains(t *testing.T) {
	t.Parallel()

	l := NewList()

	if err := l.Add("192.168.1.0/24"); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if err := l.Add("10.0.0.0/8"); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.255", true},
		{"192.168.2.1", false},
		{"10.1.2.3", true},
		{"11.0.0.1", false},
		{"not-an-ip", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := l.Contains(tt.ip); got != tt.want {
			t.Errorf("Contains(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

// TestList_InvalidCIDR проверяет, что Add возвращает ErrInvalidCIDR
// для некорректных входных данных.
func TestList_InvalidCIDR(t *testing.T) {
	t.Parallel()

	l := NewList()

	invalid := []string{
		"",
		"192.168.1.1",
		"192.168.1.0/33",
		"not-a-cidr",
		"192.168.1.0/-1",
	}

	for _, cidr := range invalid {
		err := l.Add(cidr)
		if err == nil {
			t.Errorf("Add(%q): expected error, got nil", cidr)
			continue
		}
		if !errors.Is(err, ErrInvalidCIDR) {
			t.Errorf("Add(%q): expected ErrInvalidCIDR, got %v", cidr, err)
		}
	}

	if l.Len() != 0 {
		t.Fatalf("expected empty list, got %d", l.Len())
	}
}

// TestList_Remove проверяет удаление подсети.
func TestList_Remove(t *testing.T) {
	t.Parallel()

	l := NewList()
	_ = l.Add("192.168.1.0/24")

	if l.Len() != 1 {
		t.Fatalf("expected 1 net, got %d", l.Len())
	}

	if !l.Remove("192.168.1.0/24") {
		t.Fatal("Remove should return true for existing net")
	}
	if l.Len() != 0 {
		t.Fatalf("expected 0 nets, got %d", l.Len())
	}
	if l.Remove("192.168.1.0/24") {
		t.Fatal("Remove should return false for missing net")
	}
	if l.Remove("invalid-cidr") {
		t.Fatal("Remove should return false for invalid input")
	}
}

// TestList_AddIsIdempotent проверяет, что повторное добавление
// одной и той же подсети не дублирует её.
func TestList_AddIsIdempotent(t *testing.T) {
	t.Parallel()

	l := NewList()

	_ = l.Add("192.168.1.0/24")
	_ = l.Add("192.168.1.0/24")
	_ = l.Add("192.168.1.5/24") // та же сеть после нормализации

	if l.Len() != 1 {
		t.Fatalf("expected 1 net, got %d", l.Len())
	}
}

// TestList_All проверяет получение полного списка подсетей.
func TestList_All(t *testing.T) {
	t.Parallel()

	l := NewList()
	_ = l.Add("192.168.1.0/24")
	_ = l.Add("10.0.0.0/8")

	got := l.All()
	sort.Strings(got)

	want := []string{"10.0.0.0/8", "192.168.1.0/24"}

	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %v", len(want), len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestList_SingleHost проверяет /32-подсеть (один конкретный IP).
func TestList_SingleHost(t *testing.T) {
	t.Parallel()

	l := NewList()
	_ = l.Add("192.168.1.42/32")

	if !l.Contains("192.168.1.42") {
		t.Fatal("should contain exact IP")
	}
	if l.Contains("192.168.1.43") {
		t.Fatal("should not contain neighbor IP")
	}
}

// TestList_ZeroMask проверяет /0-подсеть (весь интернет).
func TestList_ZeroMask(t *testing.T) {
	t.Parallel()

	l := NewList()
	_ = l.Add("0.0.0.0/0")

	if !l.Contains("8.8.8.8") {
		t.Fatal("0.0.0.0/0 should match any IPv4")
	}
}
