package usecase

// Attempt описывает попытку авторизации, поступившую от клиента.
type Attempt struct {
	Login    string
	Password string
	IP       string
}

// Decision — результат проверки попытки.
// Reason используется только для логирования и метрик,
// наружу отдаётся только Allowed.
type Decision struct {
	Allowed bool
	Reason  string
}

// rule — одно звено цепочки проверок (Chain of Responsibility).
// Возвращает (Decision, true), если принимает решение и останавливает цепочку.
// Возвращает (zero, false), чтобы передать управление следующему правилу.
type rule func(Attempt) (Decision, bool)

// CheckAttempt — usecase проверки попытки авторизации.
// Правила применяются в порядке: whitelist → blacklist → login → password → ip.
// Whitelist имеет высший приоритет: если IP в нём, авторизация разрешается
// даже если он же присутствует в blacklist или исчерпал лимиты.
type CheckAttempt struct {
	rules []rule
}

// NewCheckAttempt собирает цепочку правил из зависимостей.
func NewCheckAttempt(limiter Limiter, ipRules IPRuleStore) *CheckAttempt {
	return &CheckAttempt{
		rules: []rule{
			whitelistRule(ipRules),
			blacklistRule(ipRules),
			loginRule(limiter),
			passwordRule(limiter),
			ipRule(limiter),
		},
	}
}

// Execute прогоняет попытку через цепочку правил.
// Возвращает Decision с первым принятым решением.
func (uc *CheckAttempt) Execute(a Attempt) Decision {
	for _, r := range uc.rules {
		if d, stop := r(a); stop {
			return d
		}
	}

	return Decision{Allowed: true, Reason: "ok"}
}

// ---- Отдельные правила ----

func whitelistRule(store IPRuleStore) rule {
	return func(a Attempt) (Decision, bool) {
		if store.IsWhitelisted(a.IP) {
			return Decision{Allowed: true, Reason: "whitelist"}, true
		}

		return Decision{}, false
	}
}

func blacklistRule(store IPRuleStore) rule {
	return func(a Attempt) (Decision, bool) {
		if store.IsBlacklisted(a.IP) {
			return Decision{Allowed: false, Reason: "blacklist"}, true
		}

		return Decision{}, false
	}
}

func loginRule(limiter Limiter) rule {
	return func(a Attempt) (Decision, bool) {
		if !limiter.AllowByLogin(a.Login) {
			return Decision{Allowed: false, Reason: "login limit"}, true
		}

		return Decision{}, false
	}
}

func passwordRule(limiter Limiter) rule {
	return func(a Attempt) (Decision, bool) {
		if !limiter.AllowByPassword(a.Password) {
			return Decision{Allowed: false, Reason: "password limit"}, true
		}

		return Decision{}, false
	}
}

func ipRule(limiter Limiter) rule {
	return func(a Attempt) (Decision, bool) {
		if !limiter.AllowByIP(a.IP) {
			return Decision{Allowed: false, Reason: "ip limit"}, true
		}

		return Decision{}, false
	}
}
