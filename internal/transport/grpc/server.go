// Package grpcsrv реализует gRPC-транспорт сервиса анти-брутфорс.
// Пакет назван grpcsrv, чтобы не конфликтовать с google.golang.org/grpc.
package grpcsrv

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pbv1 "github.com/boba180799/anti-bruteforce/api/proto/v1"
	"github.com/boba180799/anti-bruteforce/internal/netutil"
	"github.com/boba180799/anti-bruteforce/internal/usecase"
)

// Server реализует pbv1.AntiBruteforceServiceServer.
type Server struct {
	pbv1.UnimplementedAntiBruteforceServiceServer

	check *usecase.CheckAttempt
	reset *usecase.ResetBucket
	rules *usecase.ManageRules
	log   *slog.Logger
}

// NewServer создаёт gRPC-сервер, делегирующий работу usecase-слою.
func NewServer(
	check *usecase.CheckAttempt,
	reset *usecase.ResetBucket,
	rules *usecase.ManageRules,
	log *slog.Logger,
) *Server {
	return &Server{
		check: check,
		reset: reset,
		rules: rules,
		log:   log,
	}
}

// CheckAttempt реализует RPC CheckAttempt.
func (s *Server) CheckAttempt(
	_ context.Context,
	req *pbv1.CheckAttemptRequest,
) (*pbv1.CheckAttemptResponse, error) {
	if req.GetLogin() == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if req.GetIp() == "" {
		return nil, status.Error(codes.InvalidArgument, "ip is required")
	}

	decision := s.check.Execute(usecase.Attempt{
		Login:    req.GetLogin(),
		Password: req.GetPassword(),
		IP:       req.GetIp(),
	})

	s.log.Debug("check attempt",
		"login", req.GetLogin(),
		"ip", req.GetIp(),
		"allowed", decision.Allowed,
		"reason", decision.Reason,
	)

	return &pbv1.CheckAttemptResponse{Ok: decision.Allowed}, nil
}

// ResetBucket реализует RPC ResetBucket.
func (s *Server) ResetBucket(
	_ context.Context,
	req *pbv1.ResetBucketRequest,
) (*emptypb.Empty, error) {
	if req.GetLogin() == "" && req.GetIp() == "" {
		return nil, status.Error(codes.InvalidArgument, "login or ip is required")
	}

	s.reset.Execute(req.GetLogin(), req.GetIp())

	return &emptypb.Empty{}, nil
}

// ListIpRules реализует RPC ListIpRules.
//
//nolint:revive // name must match proto-generated interface AntiBruteforceServiceServer
func (s *Server) ListIpRules(
	_ context.Context,
	req *pbv1.ListIpRulesRequest,
) (*pbv1.ListIpRulesResponse, error) {
	filter, err := ruleTypeFromPB(req.GetFilterType())
	if err != nil {
		return nil, err
	}

	rules := s.rules.List(filter)

	out := make([]*pbv1.IpRule, 0, len(rules))
	for _, r := range rules {
		out = append(out, &pbv1.IpRule{
			Cidr:        r.CIDR,
			Type:        ruleTypeToPB(r.Type),
			Description: r.Description,
		})
	}

	return &pbv1.ListIpRulesResponse{IpRules: out}, nil
}

// CreateIpRule реализует RPC CreateIpRule.
//
//nolint:revive // name must match proto-generated interface AntiBruteforceServiceServer
func (s *Server) CreateIpRule(
	_ context.Context,
	req *pbv1.CreateIpRuleRequest,
) (*pbv1.IpRule, error) {
	ruleType, err := ruleTypeFromPB(req.GetType())
	if err != nil {
		return nil, err
	}
	if ruleType == usecase.RuleTypeUnspecified {
		return nil, status.Error(codes.InvalidArgument, "rule type is required")
	}

	rule := usecase.Rule{
		CIDR:        req.GetCidr(),
		Type:        ruleType,
		Description: req.GetDescription(),
	}
	if err := s.rules.Add(rule); err != nil {
		return nil, mapRuleError(err)
	}

	return &pbv1.IpRule{
		Cidr:        req.GetCidr(),
		Type:        req.GetType(),
		Description: req.GetDescription(),
	}, nil
}

// DeleteIpRule реализует RPC DeleteIpRule.
//
//nolint:revive // name must match proto-generated interface AntiBruteforceServiceServer
func (s *Server) DeleteIpRule(
	_ context.Context,
	req *pbv1.DeleteIpRuleRequest,
) (*emptypb.Empty, error) {
	ruleType, err := ruleTypeFromPB(req.GetType())
	if err != nil {
		return nil, err
	}
	if ruleType == usecase.RuleTypeUnspecified {
		return nil, status.Error(codes.InvalidArgument, "rule type is required")
	}

	removed, err := s.rules.Remove(ruleType, req.GetCidr())
	if err != nil {
		return nil, mapRuleError(err)
	}
	if !removed {
		return nil, status.Error(codes.NotFound, "rule not found")
	}

	return &emptypb.Empty{}, nil
}

// ruleTypeFromPB переводит enum из proto в доменный тип.
func ruleTypeFromPB(t pbv1.IpRuleType) (usecase.RuleType, error) {
	switch t {
	case pbv1.IpRuleType_IP_RULE_TYPE_UNSPECIFIED:
		return usecase.RuleTypeUnspecified, nil
	case pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST:
		return usecase.RuleTypeWhitelist, nil
	case pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST:
		return usecase.RuleTypeBlacklist, nil
	}

	return 0, status.Errorf(codes.InvalidArgument, "unknown rule type: %v", t)
}

// ruleTypeToPB переводит доменный тип в enum proto.
func ruleTypeToPB(t usecase.RuleType) pbv1.IpRuleType {
	switch t {
	case usecase.RuleTypeWhitelist:
		return pbv1.IpRuleType_IP_RULE_TYPE_WHITELIST
	case usecase.RuleTypeBlacklist:
		return pbv1.IpRuleType_IP_RULE_TYPE_BLACKLIST
	case usecase.RuleTypeUnspecified:
		return pbv1.IpRuleType_IP_RULE_TYPE_UNSPECIFIED
	}

	return pbv1.IpRuleType_IP_RULE_TYPE_UNSPECIFIED
}

// mapRuleError переводит доменную ошибку в gRPC-статус.
func mapRuleError(err error) error {
	if errors.Is(err, netutil.ErrInvalidCIDR) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, usecase.ErrInvalidRuleType) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return status.Error(codes.Internal, err.Error())
}
