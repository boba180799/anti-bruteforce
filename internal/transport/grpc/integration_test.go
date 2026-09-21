// Package grpcsrv_test содержит интеграционные тесты gRPC-транспорта.
package grpcsrv_test

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
	"github.com/boba180799/anti-bruteforce/internal/limiter"
	"github.com/boba180799/anti-bruteforce/internal/repository/memory"
	grpcsrv "github.com/boba180799/anti-bruteforce/internal/transport/grpc"
	"github.com/boba180799/anti-bruteforce/internal/usecase"
)

const (
	testLogin         = "alice"
	testPassword      = "secret"
	testIP            = "1.2.3.4"
	testCIDRWhitelist = "10.0.0.0/8"
	testCIDRBlacklist = "6.6.6.0/24"
)

// newTestClient поднимает in-memory gRPC-сервер и возвращает клиента.
// Возвращает клиент и функцию очистки.
func newTestClient(t *testing.T, loginLimit, ipLimit float64) (pbv1.AntiBruteforceServiceClient, func()) {
	t.Helper()

	log := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))

	storage := limiter.NewStorage(limiter.StorageConfig{
		LoginCapacity:    loginLimit,
		LoginRate:        0,
		PasswordCapacity: 1000,
		PasswordRate:     0,
		IPCapacity:       ipLimit,
		IPRate:           0,
		TTL:              time.Minute,
	})
	rules := memory.NewIPRules()

	checkUC := usecase.NewCheckAttempt(storage, rules)
	resetUC := usecase.NewResetBucket(storage)
	rulesUC := usecase.NewManageRules(rules)

	grpcServer := grpc.NewServer()
	pbv1.RegisterAntiBruteforceServiceServer(
		grpcServer,
		grpcsrv.NewServer(checkUC, resetUC, rulesUC, log),
	)

	lis := bufconn.Listen(1024 * 1024)
	go func() {
		_ = grpcServer.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}

	client := pbv1.NewAntiBruteforceServiceClient(conn)

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
	}

	return client, cleanup
}

func TestIntegration_CheckAttempt_Allowed(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 10, 1000)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    testLogin,
		Password: testPassword,
		Ip:       testIP,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetOk() {
		t.Fatal("expected allowed")
	}
}

func TestIntegration_CheckAttempt_LoginLimit(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 3, 1000)
	defer cleanup()

	ctx := context.Background()

	for i := range 3 {
		resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
			Login:    testLogin,
			Password: testPassword,
			Ip:       testIP,
		})
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if !resp.GetOk() {
			t.Fatalf("attempt %d should be allowed", i)
		}
	}

	resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    testLogin,
		Password: testPassword,
		Ip:       testIP,
	})
	if err != nil {
		t.Fatalf("4th attempt: %v", err)
	}
	if resp.GetOk() {
		t.Fatal("4th attempt should be denied")
	}
}

func TestIntegration_CheckAttempt_IPLimit(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 1000, 3)
	defer cleanup()

	ctx := context.Background()

	// Исчерпаем лимит по IP (3 попытки), варьируя логин, чтобы не задеть login-limit.
	for i := range 3 {
		resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
			Login:    "user" + string(rune('a'+i)),
			Password: testPassword,
			Ip:       testIP,
		})
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if !resp.GetOk() {
			t.Fatalf("attempt %d should be allowed", i)
		}
	}

	// 4-я попытка с другим логином, но тем же IP — должна быть заблокирована.
	resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    "newuser",
		Password: testPassword,
		Ip:       testIP,
	})
	if err != nil {
		t.Fatalf("4th attempt: %v", err)
	}
	if resp.GetOk() {
		t.Fatal("4th attempt should be denied by IP limit")
	}
}

func TestIntegration_CheckAttempt_WhitelistBeatsLimits(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 1, 1000)
	defer cleanup()

	ctx := context.Background()

	// Добавляем в whitelist.
	_, err := client.CreateIpRule(ctx, &pbv1.CreateIpRuleRequest{
		Cidr: testCIDRWhitelist,
		Type: pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST,
	})
	if err != nil {
		t.Fatalf("create whitelist: %v", err)
	}

	// Пробиваем лимит по логину.
	for range 2 {
		_, _ = client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
			Login:    testLogin,
			Password: testPassword,
			Ip:       testIP,
		})
	}

	// С whitelist-IP — должно проходить, несмотря на исчерпанный лимит логина.
	resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    testLogin,
		Password: testPassword,
		Ip:       "10.1.2.3",
	})
	if err != nil {
		t.Fatalf("whitelist attempt: %v", err)
	}
	if !resp.GetOk() {
		t.Fatal("whitelist should override limits")
	}
}

func TestIntegration_CheckAttempt_Blacklist(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 1000, 1000)
	defer cleanup()

	ctx := context.Background()

	_, err := client.CreateIpRule(ctx, &pbv1.CreateIpRuleRequest{
		Cidr: testCIDRBlacklist,
		Type: pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST,
	})
	if err != nil {
		t.Fatalf("create blacklist: %v", err)
	}

	resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    testLogin,
		Password: testPassword,
		Ip:       "6.6.6.1",
	})
	if err != nil {
		t.Fatalf("blacklist attempt: %v", err)
	}
	if resp.GetOk() {
		t.Fatal("blacklist should deny")
	}
}

func TestIntegration_ResetBucket(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 2, 1000)
	defer cleanup()

	ctx := context.Background()

	// Исчерпаем лимит.
	for range 3 {
		_, _ = client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
			Login:    testLogin,
			Password: testPassword,
			Ip:       testIP,
		})
	}

	// Сброс.
	_, err := client.ResetBucket(ctx, &pbv1.ResetBucketRequest{Login: testLogin})
	if err != nil {
		t.Fatalf("reset: %v", err)
	}

	// Снова должно пускать.
	resp, err := client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    testLogin,
		Password: testPassword,
		Ip:       testIP,
	})
	if err != nil {
		t.Fatalf("after reset: %v", err)
	}
	if !resp.GetOk() {
		t.Fatal("after reset should be allowed")
	}
}

func TestIntegration_IpRuleCRUD(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 1000, 1000)
	defer cleanup()

	ctx := context.Background()

	// Create.
	_, err := client.CreateIpRule(ctx, &pbv1.CreateIpRuleRequest{
		Cidr:        testCIDRWhitelist,
		Type:        pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST,
		Description: "office",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// List.
	resp, err := client.ListIpRules(ctx, &pbv1.ListIpRulesRequest{
		FilterType: pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(resp.GetIpRules()) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(resp.GetIpRules()))
	}

	// Delete.
	_, err = client.DeleteIpRule(ctx, &pbv1.DeleteIpRuleRequest{
		Cidr: testCIDRWhitelist,
		Type: pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST,
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify empty.
	resp, _ = client.ListIpRules(ctx, &pbv1.ListIpRulesRequest{
		FilterType: pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST,
	})
	if len(resp.GetIpRules()) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(resp.GetIpRules()))
	}
}

func TestIntegration_Validation_EmptyFields(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 10, 1000)
	t.Cleanup(cleanup)

	cases := []struct {
		name string
		req  *pbv1.CheckAttemptRequest
	}{
		{"no login", &pbv1.CheckAttemptRequest{Password: "x", Ip: testIP}},
		{"no password", &pbv1.CheckAttemptRequest{Login: testLogin, Ip: testIP}},
		{"no ip", &pbv1.CheckAttemptRequest{Login: testLogin, Password: "x"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			_, err := client.CheckAttempt(ctx, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if code := status.Code(err); code != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", code)
			}
		})
	}
}

func TestIntegration_DeleteIpRule_NotFound(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, 1000, 1000)
	defer cleanup()

	ctx := context.Background()

	_, err := client.DeleteIpRule(ctx, &pbv1.DeleteIpRuleRequest{
		Cidr: testCIDRWhitelist,
		Type: pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST,
	})
	if err == nil {
		t.Fatal("expected NotFound error")
	}
	if code := status.Code(err); code != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", code)
	}
}
