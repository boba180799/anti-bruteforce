package usecase

import "errors"

// ErrInvalidRuleType возвращается при неизвестном типе правила.
var ErrInvalidRuleType = errors.New("invalid rule type")

// Rule — правило для добавления через usecase.
type Rule struct {
	CIDR        string
	Type        RuleType
	Description string
}

// ManageRules — usecase управления whitelist/blacklist.
type ManageRules struct {
	store IPRuleStore
}

// NewManageRules создаёт usecase управления правилами.
func NewManageRules(store IPRuleStore) *ManageRules {
	return &ManageRules{store: store}
}

// Add добавляет правило указанного типа.
func (uc *ManageRules) Add(rule Rule) error {
	switch rule.Type {
	case RuleTypeWhitelist:
		return uc.store.AddWhitelist(rule.CIDR, rule.Description)
	case RuleTypeBlacklist:
		return uc.store.AddBlacklist(rule.CIDR, rule.Description)
	case RuleTypeUnspecified:
		return ErrInvalidRuleType
	}

	return ErrInvalidRuleType
}

// Remove удаляет правило.
func (uc *ManageRules) Remove(ruleType RuleType, cidr string) (bool, error) {
	switch ruleType {
	case RuleTypeWhitelist:
		return uc.store.RemoveWhitelist(cidr), nil
	case RuleTypeBlacklist:
		return uc.store.RemoveBlacklist(cidr), nil
	case RuleTypeUnspecified:
		return false, ErrInvalidRuleType
	}

	return false, ErrInvalidRuleType
}

// List возвращает правила по фильтру.
// RuleTypeUnspecified — вернуть все правила.
func (uc *ManageRules) List(filterType RuleType) []Rule {
	var out []Rule

	if filterType == RuleTypeUnspecified || filterType == RuleTypeWhitelist {
		for _, info := range uc.store.ListWhitelist() {
			out = append(out, Rule{
				CIDR:        info.CIDR,
				Type:        RuleTypeWhitelist,
				Description: info.Description,
			})
		}
	}

	if filterType == RuleTypeUnspecified || filterType == RuleTypeBlacklist {
		for _, info := range uc.store.ListBlacklist() {
			out = append(out, Rule{
				CIDR:        info.CIDR,
				Type:        RuleTypeBlacklist,
				Description: info.Description,
			})
		}
	}

	return out
}
