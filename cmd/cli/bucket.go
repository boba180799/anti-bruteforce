package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
)

// newBucketCmd собирает группу команд `bucket`.
func newBucketCmd(server *string, timeout *time.Duration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bucket",
		Short: "Управление вёдрами лимитов",
	}

	cmd.AddCommand(newBucketResetCmd(server, timeout))

	return cmd
}

// newBucketResetCmd — команда `bucket reset`.
func newBucketResetCmd(server *string, timeout *time.Duration) *cobra.Command {
	var (
		login string
		ip    string
	)

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Сбросить ведро для логина и/или IP",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if login == "" && ip == "" {
				return fmt.Errorf("укажите хотя бы один из флагов: --login или --ip")
			}

			ctx, cancel := withTimeout(cmd.Context(), *timeout)
			defer cancel()

			client, cleanup, err := dialClient(*server)
			if err != nil {
				return err
			}
			defer cleanup()

			_, err = client.ResetBucket(ctx, &pbv1.ResetBucketRequest{
				Login: login,
				Ip:    ip,
			})
			if err != nil {
				return fmt.Errorf("reset bucket: %w", err)
			}

			cmd.Println("bucket reset ok")

			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "логин")
	cmd.Flags().StringVar(&ip, "ip", "", "IP-адрес")

	return cmd
}
