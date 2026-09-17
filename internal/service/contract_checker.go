package service

import "context"

type ContractChecker interface {
	CheckService(ctx context.Context, companyID, serviceCode string) (CheckResult, error)
}

type CheckResult struct {
	Allowed bool
	Reason  string
}

//go:generate mockgen -source=contract_checker.go -destination=mocks/contract_checker.go -package=mocks
