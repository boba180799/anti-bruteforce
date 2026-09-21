package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
)

// newWhitelistCmd собирает группу команд `whitelist`.
func newWhitelistCmd(server *string, timeout *time.Duration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Управление whitelist-правилами",
	}

	cmd.AddCommand(
		newRulesAddCmd(server, timeout, pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST),
		newRulesRemoveCmd(server, timeout, pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST),
		newRulesListCmd(server, timeout, pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST),
	)

	return cmd
}

// newBlacklistCmd собирает группу команд `blacklist`.
func newBlacklistCmd(server *string, timeout *time.Duration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blacklist",
		Short: "Управление blacklist-правилами",
	}

	cmd.AddCommand(
		newRulesAddCmd(server, timeout, pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST),
		newRulesRemoveCmd(server, timeout, pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST),
		newRulesListCmd(server, timeout, pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST),
	)

	return cmd
}

func newRulesAddCmd(server *string, timeout *time.Duration, ruleType pbv1.IpRuleType) *cobra.Command {
	var (
		cidr        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить правило",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cidr == "" {
				return fmt.Errorf("флаг --cidr обязателен")
			}

			ctx, cancel := withTimeout(cmd.Context(), *timeout)
			defer cancel()

			client, cleanup, err := dialClient(*server)
			if err != nil {
				return err
			}
			defer cleanup()

			rule, err := client.CreateIpRule(ctx, &pbv1.CreateIpRuleRequest{
				Cidr:        cidr,
				Type:        ruleType,
				Description: description,
			})
			if err != nil {
				return fmt.Errorf("create rule: %w", err)
			}

			cmd.Printf("added %s rule: %s\n", ruleTypeLabel(ruleType), rule.GetCidr())

			return nil
		},
	}

	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR-подсеть (например, 10.0.0.0/8)")
	cmd.Flags().StringVar(&description, "description", "", "описание правила")

	return cmd
}

func newRulesRemoveCmd(server *string, timeout *time.Duration, ruleType pbv1.IpRuleType) *cobra.Command {
	var cidr string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Удалить правило",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cidr == "" {
				return fmt.Errorf("флаг --cidr обязателен")
			}

			ctx, cancel := withTimeout(cmd.Context(), *timeout)
			defer cancel()

			client, cleanup, err := dialClient(*server)
			if err != nil {
				return err
			}
			defer cleanup()

			_, err = client.DeleteIpRule(ctx, &pbv1.DeleteIpRuleRequest{
				Cidr: cidr,
				Type: ruleType,
			})
			if err != nil {
				return fmt.Errorf("delete rule: %w", err)
			}

			cmd.Printf("removed %s rule: %s\n", ruleTypeLabel(ruleType), cidr)

			return nil
		},
	}

	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR-подсеть")

	return cmd
}

func newRulesListCmd(server *string, timeout *time.Duration, ruleType pbv1.IpRuleType) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Показать все правила",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := withTimeout(cmd.Context(), *timeout)
			defer cancel()

			client, cleanup, err := dialClient(*server)
			if err != nil {
				return err
			}
			defer cleanup()

			resp, err := client.ListIpRules(ctx, &pbv1.ListIpRulesRequest{
				FilterType: ruleType,
			})
			if err != nil {
				return fmt.Errorf("list rules: %w", err)
			}

			rules := resp.GetIpRules()
			sort.Slice(rules, func(i, j int) bool { return rules[i].GetCidr() < rules[j].GetCidr() })

			if len(rules) == 0 {
				cmd.Println("(no rules)")
				return nil
			}

			for _, r := range rules {
				cmd.Printf("%-18s %s\n", r.GetCidr(), r.GetDescription())
			}

			return nil
		},
	}

	return cmd
}

// ruleTypeLabel возвращает человекочитаемое имя типа правила.
func ruleTypeLabel(t pbv1.IpRuleType) string {
	switch t {
	case pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST:
		return "whitelist"
	case pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST:
		return "blacklist"
	case pbv1.IpRuleType_IP_RULE_TYPE_UNSPECIFIED:
		return "unknown"
	}

	return "unknown"
}
