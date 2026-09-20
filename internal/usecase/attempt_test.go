package usecase

import (
	"testing"
)

const (
	// testIP — типичный IP-адрес клиента, используемый в тестах.
	testIP = "1.2.3.4"

	// testLogin — типичный логин, используемый в тестах.
	testLogin = "alice"

	// testPassword — типичный пароль, используемый в тестах.
	testPassword = "secret"
)

// ---- Ручные моки ----
//
// Мы не тянем mockgen или testify — обходимся простыми структурами.
// Каждое поле — необязательная функция, которая задаёт поведение метода.
// Если функция nil, действует значение по умолчанию.

type mockLimiter struct {
	allowLogin    func(string) bool
	allowPassword func(string) bool
	allowIP       func(string) bool

	resetLoginCalls []string
	resetIPCalls    []string
}

func (m *mockLimiter) AllowByLogin(s string) bool {
	if m.allowLogin == nil {
		return true
	}

	return m.allowLogin(s)
}

func (m *mockLimiter) AllowByPassword(s string) bool {
	if m.allowPassword == nil {
		return true
	}

	return m.allowPassword(s)
}

func (m *mockLimiter) AllowByIP(s string) bool {
	if m.allowIP == nil {
		return true
	}

	return m.allowIP(s)
}

func (m *mockLimiter) ResetLogin(s string) { m.resetLoginCalls = append(m.resetLoginCalls, s) }
func (m *mockLimiter) ResetIP(s string)    { m.resetIPCalls = append(m.resetIPCalls, s) }

type mockIPRuleStore struct {
	whitelisted func(string) bool
	blacklisted func(string) bool
}

func (m *mockIPRuleStore) IsWhitelisted(ip string) bool {
	if m.whitelisted == nil {
		return false
	}

	return m.whitelisted(ip)
}

func (m *mockIPRuleStore) IsBlacklisted(ip string) bool {
	if m.blacklisted == nil {
		return false
	}

	return m.blacklisted(ip)
}

func (m *mockIPRuleStore) AddWhitelist(string) error   { return nil }
func (m *mockIPRuleStore) RemoveWhitelist(string) bool { return true }
func (m *mockIPRuleStore) AddBlacklist(string) error   { return nil }
func (m *mockIPRuleStore) RemoveBlacklist(string) bool { return true }
func (m *mockIPRuleStore) ListWhitelist() []string     { return nil }
func (m *mockIPRuleStore) ListBlacklist() []string     { return nil }

// ---- Тесты ----

func TestCheckAttempt_AllowsWhenEverythingOk(t *testing.T) {
	t.Parallel()

	uc := NewCheckAttempt(&mockLimiter{}, &mockIPRuleStore{})

	d := uc.Execute(Attempt{Login: testLogin, Password: testPassword, IP: testIP})
	if !d.Allowed {
		t.Fatalf("expected allowed, got denied: %s", d.Reason)
	}
}

func TestCheckAttempt_WhitelistAllows(t *testing.T) {
	t.Parallel()

	limiter := &mockLimiter{
		allowLogin: func(string) bool { return false }, // лимиты исчерпаны
	}
	rules := &mockIPRuleStore{
		whitelisted: func(ip string) bool { return ip == "10.0.0.1" },
	}

	uc := NewCheckAttempt(limiter, rules)

	d := uc.Execute(Attempt{Login: testLogin, Password: "x", IP: "10.0.0.1"})
	if !d.Allowed {
		t.Fatalf("whitelist should override limits, got: %s", d.Reason)
	}
}

func TestCheckAttempt_WhitelistBeatsBlacklist(t *testing.T) {
	t.Parallel()

	rules := &mockIPRuleStore{
		whitelisted: func(string) bool { return true },
		blacklisted: func(string) bool { return true },
	}

	uc := NewCheckAttempt(&mockLimiter{}, rules)

	d := uc.Execute(Attempt{Login: testLogin, Password: "x", IP: testIP})
	if !d.Allowed {
		t.Fatalf("whitelist should win over blacklist, got: %s", d.Reason)
	}
	if d.Reason != "whitelist" {
		t.Fatalf("expected reason=whitelist, got %q", d.Reason)
	}
}

func TestCheckAttempt_BlacklistDenies(t *testing.T) {
	t.Parallel()

	rules := &mockIPRuleStore{
		blacklisted: func(ip string) bool { return ip == "6.6.6.6" },
	}

	uc := NewCheckAttempt(&mockLimiter{}, rules)

	d := uc.Execute(Attempt{Login: testLogin, Password: "x", IP: "6.6.6.6"})
	if d.Allowed {
		t.Fatal("blacklist should deny")
	}
	if d.Reason != "blacklist" {
		t.Fatalf("expected reason=blacklist, got %q", d.Reason)
	}
}

func TestCheckAttempt_LoginLimitDenies(t *testing.T) {
	t.Parallel()

	limiter := &mockLimiter{
		allowLogin: func(string) bool { return false },
	}

	uc := NewCheckAttempt(limiter, &mockIPRuleStore{})

	d := uc.Execute(Attempt{Login: testLogin, Password: "x", IP: testIP})
	if d.Allowed {
		t.Fatal("login limit should deny")
	}
	if d.Reason != "login limit" {
		t.Fatalf("expected reason=login limit, got %q", d.Reason)
	}
}

func TestCheckAttempt_PasswordLimitDenies(t *testing.T) {
	t.Parallel()

	limiter := &mockLimiter{
		allowPassword: func(string) bool { return false },
	}

	uc := NewCheckAttempt(limiter, &mockIPRuleStore{})

	d := uc.Execute(Attempt{Login: testLogin, Password: "x", IP: testIP})
	if d.Allowed || d.Reason != "password limit" {
		t.Fatalf("expected password limit denial, got %+v", d)
	}
}

func TestCheckAttempt_IPLimitDenies(t *testing.T) {
	t.Parallel()

	limiter := &mockLimiter{
		allowIP: func(string) bool { return false },
	}

	uc := NewCheckAttempt(limiter, &mockIPRuleStore{})

	d := uc.Execute(Attempt{Login: testLogin, Password: "x", IP: testIP})
	if d.Allowed || d.Reason != "ip limit" {
		t.Fatalf("expected ip limit denial, got %+v", d)
	}
}

func TestResetBucket_ResetsLoginAndIP(t *testing.T) {
	t.Parallel()

	limiter := &mockLimiter{}
	uc := NewResetBucket(limiter)

	uc.Execute(testLogin, testIP)

	if len(limiter.resetLoginCalls) != 1 || limiter.resetLoginCalls[0] != testLogin {
		t.Fatalf("expected login reset, got %v", limiter.resetLoginCalls)
	}
	if len(limiter.resetIPCalls) != 1 || limiter.resetIPCalls[0] != testIP {
		t.Fatalf("expected ip reset, got %v", limiter.resetIPCalls)
	}
}

func TestResetBucket_SkipsEmptyFields(t *testing.T) {
	t.Parallel()

	limiter := &mockLimiter{}
	uc := NewResetBucket(limiter)

	uc.Execute("", testIP)

	if len(limiter.resetLoginCalls) != 0 {
		t.Fatalf("empty login should be skipped, got %v", limiter.resetLoginCalls)
	}
	if len(limiter.resetIPCalls) != 1 {
		t.Fatalf("expected ip reset, got %v", limiter.resetIPCalls)
	}
}
