package integration_test

import (
	"github.com/allora-network/allora-chain/app/upgrades/v0_15_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_16_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_17_0"
	"github.com/allora-network/allora-chain/app/upgrades/v0_18_0"
	testCommon "github.com/allora-network/allora-chain/test/common"
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
	// BaselineAppVersion is the version every validator must report before the upgrade. Empty skips the check.
	BaselineAppVersion string
	// BaselineModuleVersions are the consensus versions the chain must report before the upgrade.
	BaselineModuleVersions map[string]uint64
	// NewStateChecks builds the checks that seed the baseline and verify migrated state. Nil skips them.
	NewStateChecks func() upgradeStateChecks
}

// upgradeStateChecks seeds a baseline chain and verifies what an upgrade does to that state.
type upgradeStateChecks interface {
	// SeedBaseline populates the baseline chain through transactions.
	SeedBaseline(m testCommon.TestConfig)
	// ChoosePlanHeight returns the plan height to propose, at or after earliest.
	ChoosePlanHeight(m testCommon.TestConfig, earliest int64) int64
	// BeforeUpgradeBlock runs once the proposal has passed and snapshots the last baseline height.
	BeforeUpgradeBlock(m testCommon.TestConfig, planHeight int64)
	// VerifyMigration asserts migrated and preserved values at the upgrade block.
	VerifyMigration(m testCommon.TestConfig, planHeight int64)
	// DriveMigratedState runs activity on the migrated state and checks the validators agree.
	DriveMigratedState(m testCommon.TestConfig, nodes []validatorNode, planHeight int64)
}

const DefaultPlanHeightDelta = int64(50)

var DefaultUpgradeTarget = v0_18_0.UpgradeName

var UpgradeScenarios = map[string]UpgradeScenario{
	v0_15_0.UpgradeName: {
		UpgradeName:            v0_15_0.UpgradeName,
		PlanHeightDelta:        DefaultPlanHeightDelta,
		BaselineAppVersion:     "",
		BaselineModuleVersions: nil,
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
		UpgradeName:            v0_16_0.UpgradeName,
		PlanHeightDelta:        DefaultPlanHeightDelta,
		BaselineAppVersion:     "",
		BaselineModuleVersions: nil,
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
		UpgradeName:            v0_17_0.UpgradeName,
		PlanHeightDelta:        DefaultPlanHeightDelta,
		BaselineAppVersion:     "",
		BaselineModuleVersions: nil,
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
	v0_18_0.UpgradeName: {
		UpgradeName:        v0_18_0.UpgradeName,
		PlanHeightDelta:    DefaultPlanHeightDelta,
		BaselineAppVersion: "v0.17.2",
		BaselineModuleVersions: map[string]uint64{
			"scheduler": 1,
			"emissions": 15,
			"mint":      6,
		},
		NewStateChecks: func() upgradeStateChecks { return newPopulatedStateChecks() },
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
					version := uint64(16)
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
