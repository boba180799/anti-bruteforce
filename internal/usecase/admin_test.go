package usecase

import (
	"errors"
	"testing"
)

const (
	// testCIDRWhitelist — типичная подсеть для whitelist в тестах.
	testCIDRWhitelist = "10.0.0.0/8"

	// testCIDRBlacklist — типичная подсеть для blacklist в тестах.
	testCIDRBlacklist = "6.6.6.0/24"

	// testDescriptionOffice — описание для тестирования офисных правил.
	testDescriptionOffice = "office"
)

func TestManageRules_AddWhitelist(t *testing.T) {
	t.Parallel()

	var gotCIDR, gotDesc string
	store := &mockIPRuleStoreWithHooks{}
	store.addWhitelist = func(cidr, desc string) error {
		gotCIDR, gotDesc = cidr, desc
		return nil
	}

	uc := NewManageRules(store)

	err := uc.Add(Rule{CIDR: testCIDRWhitelist, Type: RuleTypeWhitelist, Description: testDescriptionOffice})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCIDR != testCIDRWhitelist || gotDesc != testDescriptionOffice {
		t.Fatalf("unexpected cidr/desc: %q/%q", gotCIDR, gotDesc)
	}
}

func TestManageRules_AddBlacklist(t *testing.T) {
	t.Parallel()

	var called bool
	store := &mockIPRuleStoreWithHooks{}
	store.addBlacklist = func(string, string) error {
		called = true
		return nil
	}

	uc := NewManageRules(store)

	if err := uc.Add(Rule{CIDR: testCIDRBlacklist, Type: RuleTypeBlacklist}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("blacklist add was not called")
	}
}

func TestManageRules_AddInvalidType(t *testing.T) {
	t.Parallel()

	uc := NewManageRules(&mockIPRuleStoreWithHooks{})

	err := uc.Add(Rule{CIDR: "1.2.3.0/24", Type: RuleTypeUnspecified})
	if !errors.Is(err, ErrInvalidRuleType) {
		t.Fatalf("expected ErrInvalidRuleType, got %v", err)
	}
}

func TestManageRules_Remove(t *testing.T) {
	t.Parallel()

	store := &mockIPRuleStoreWithHooks{}
	store.removeWhitelist = func(cidr string) bool { return cidr == testCIDRWhitelist }
	store.removeBlacklist = func(cidr string) bool { return cidr == testCIDRBlacklist }

	uc := NewManageRules(store)

	ok, err := uc.Remove(RuleTypeWhitelist, testCIDRWhitelist)
	if err != nil || !ok {
		t.Fatalf("whitelist remove failed: ok=%v err=%v", ok, err)
	}

	ok, err = uc.Remove(RuleTypeBlacklist, testCIDRBlacklist)
	if err != nil || !ok {
		t.Fatalf("blacklist remove failed: ok=%v err=%v", ok, err)
	}

	ok, _ = uc.Remove(RuleTypeWhitelist, "nope")
	if ok {
		t.Fatal("expected false for missing rule")
	}
}

func TestManageRules_RemoveInvalidType(t *testing.T) {
	t.Parallel()

	uc := NewManageRules(&mockIPRuleStoreWithHooks{})

	if _, err := uc.Remove(RuleTypeUnspecified, "x"); !errors.Is(err, ErrInvalidRuleType) {
		t.Fatalf("expected ErrInvalidRuleType, got %v", err)
	}
}

func TestManageRules_List(t *testing.T) {
	t.Parallel()

	store := &mockIPRuleStoreWithHooks{}
	store.listWhitelist = func() []RuleInfo {
		return []RuleInfo{{CIDR: testCIDRWhitelist, Description: testDescriptionOffice}}
	}
	store.listBlacklist = func() []RuleInfo {
		return []RuleInfo{{CIDR: testCIDRBlacklist, Description: "bad"}}
	}

	uc := NewManageRules(store)

	if got := uc.List(RuleTypeUnspecified); len(got) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(got))
	}
	if got := uc.List(RuleTypeWhitelist); len(got) != 1 || got[0].Type != RuleTypeWhitelist {
		t.Fatalf("unexpected whitelist: %+v", got)
	}
	if got := uc.List(RuleTypeBlacklist); len(got) != 1 || got[0].Type != RuleTypeBlacklist {
		t.Fatalf("unexpected blacklist: %+v", got)
	}
}

// mockIPRuleStoreWithHooks — расширенный mock для тестов ManageRules.
type mockIPRuleStoreWithHooks struct {
	mockIPRuleStore

	addWhitelist    func(cidr, desc string) error
	addBlacklist    func(cidr, desc string) error
	removeWhitelist func(cidr string) bool
	removeBlacklist func(cidr string) bool
	listWhitelist   func() []RuleInfo
	listBlacklist   func() []RuleInfo
}

func (m *mockIPRuleStoreWithHooks) AddWhitelist(cidr, desc string) error {
	if m.addWhitelist == nil {
		return nil
	}
	return m.addWhitelist(cidr, desc)
}

func (m *mockIPRuleStoreWithHooks) AddBlacklist(cidr, desc string) error {
	if m.addBlacklist == nil {
		return nil
	}
	return m.addBlacklist(cidr, desc)
}

func (m *mockIPRuleStoreWithHooks) RemoveWhitelist(cidr string) bool {
	if m.removeWhitelist == nil {
		return false
	}
	return m.removeWhitelist(cidr)
}

func (m *mockIPRuleStoreWithHooks) RemoveBlacklist(cidr string) bool {
	if m.removeBlacklist == nil {
		return false
	}
	return m.removeBlacklist(cidr)
}

func (m *mockIPRuleStoreWithHooks) ListWhitelist() []RuleInfo {
	if m.listWhitelist == nil {
		return nil
	}
	return m.listWhitelist()
}

func (m *mockIPRuleStoreWithHooks) ListBlacklist() []RuleInfo {
	if m.listBlacklist == nil {
		return nil
	}
	return m.listBlacklist()
}
