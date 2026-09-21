package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
)

// newCheckCmd — команда `check`: проверить попытку авторизации.
func newCheckCmd(server *string, timeout *time.Duration) *cobra.Command {
	var (
		login    string
		password string
		ip       string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Проверить попытку авторизации",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if login == "" || password == "" || ip == "" {
				return fmt.Errorf("нужны все три флага: --login, --password, --ip")
			}

			ctx, cancel := withTimeout(cmd.Context(), *timeout)
			defer cancel()

			client, cleanup, err := dialClient(*server)
			if err != nil {
				return err
			}
			defer cleanup()

			resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
				Login:    login,
				Password: password,
				Ip:       ip,
			})
			if err != nil {
				return fmt.Errorf("check attempt: %w", err)
			}

			if resp.GetOk() {
				cmd.Println("ALLOWED")
			} else {
				cmd.Println("DENIED")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "логин")
	cmd.Flags().StringVar(&password, "password", "", "пароль")
	cmd.Flags().StringVar(&ip, "ip", "", "IP-адрес")

	return cmd
}
