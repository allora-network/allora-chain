package integration_test

import (
	"github.com/allora-network/allora-chain/app/upgrades/v0_15_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_16_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_17_0"
)

// ModuleCheckType denotes the kind of assertion that should be applied to a module's consensus version.
type ModuleCheckType string

const (
	ModuleCheckExpectIncrease ModuleCheckType = "expect_increase"
	ModuleCheckExpectEqual    ModuleCheckType = "expect_equal"
	ModuleCheckExpectNew      ModuleCheckType = "expect_new"
)

// ModuleCheck configures how a module's consensus version should behave across an upgrade.
type ModuleCheck struct {
	ModuleName      string
	CheckType       ModuleCheckType
	ExpectedVersion *uint64
}

// UpgradeScenario captures all expectations for a given software upgrade.
type UpgradeScenario struct {
	UpgradeName     string
	PlanHeightDelta int64
	ModuleChecks    []ModuleCheck
}

const DefaultPlanHeightDelta = int64(50)

var DefaultUpgradeTarget = v0_17_0.UpgradeName

var UpgradeScenarios = map[string]UpgradeScenario{
	v0_15_0.UpgradeName: {
		UpgradeName:     v0_15_0.UpgradeName,
		PlanHeightDelta: DefaultPlanHeightDelta,
		ModuleChecks: []ModuleCheck{
			{
				ModuleName: "scheduler",
				CheckType:  ModuleCheckExpectNew,
				ExpectedVersion: func() *uint64 {
					version := uint64(1)
					return &version
				}(),
			},
			{
				ModuleName: "emissions",
				CheckType:  ModuleCheckExpectIncrease,
				ExpectedVersion: func() *uint64 {
					version := uint64(13)
					return &version
				}(),
			},
			{
				ModuleName: "mint",
				CheckType:  ModuleCheckExpectEqual,
				ExpectedVersion: func() *uint64 {
					version := uint64(6)
					return &version
				}(),
			},
		},
	},
	v0_16_0.UpgradeName: {
		UpgradeName:     v0_16_0.UpgradeName,
		PlanHeightDelta: DefaultPlanHeightDelta,
		ModuleChecks: []ModuleCheck{
			{
				ModuleName: "scheduler",
				CheckType:  ModuleCheckExpectEqual,
				ExpectedVersion: func() *uint64 {
					version := uint64(1)
					return &version
				}(),
			},
			{
				ModuleName: "emissions",
				CheckType:  ModuleCheckExpectIncrease,
				ExpectedVersion: func() *uint64 {
					version := uint64(14)
					return &version
				}(),
			},
			{
				ModuleName: "mint",
				CheckType:  ModuleCheckExpectEqual,
				ExpectedVersion: func() *uint64 {
					version := uint64(6)
					return &version
				}(),
			},
		},
	},
	v0_17_0.UpgradeName: {
		UpgradeName:     v0_17_0.UpgradeName,
		PlanHeightDelta: DefaultPlanHeightDelta,
		ModuleChecks: []ModuleCheck{
			{
				ModuleName: "scheduler",
				CheckType:  ModuleCheckExpectEqual,
				ExpectedVersion: func() *uint64 {
					version := uint64(1)
					return &version
				}(),
			},
			{
				ModuleName: "emissions",
				CheckType:  ModuleCheckExpectIncrease,
				ExpectedVersion: func() *uint64 {
					version := uint64(15)
					return &version
				}(),
			},
			{
				ModuleName: "mint",
				CheckType:  ModuleCheckExpectEqual,
				ExpectedVersion: func() *uint64 {
					version := uint64(6)
					return &version
				}(),
			},
		},
	},
}
