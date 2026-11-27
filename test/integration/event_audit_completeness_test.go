package integration_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

// ruleFunction names a function of the audit sources that checks events.
type ruleFunction struct {
	file string
	name string
}

// auditRuleFunctions are the only functions whose event-type switch cases and
// type assertions make an event "audited": the auditEvent dispatch, the rule
// functions that take events outside it, and the block-level ledger rules.
// Helpers that only order, locate or length-check events do not belong here.
// TestAuditRuleFunctionsExist fails when a listed function is gone.
var auditRuleFunctions = []ruleFunction{
	{"event_audit_test.go", "auditEvent"},
	{"event_audit_test.go", "auditTopicLifecycle"},
	{"event_audit_test.go", "auditStakeLedger"},
	{"event_audit_test.go", "expectedLedgerEvents"},
	{"event_audit_test.go", "auditTopicWeights"},
	{"event_payload_audit_test.go", "auditRewardPayouts"},
}

// eventPackages maps the Go package alias used in the audit sources to the
// proto package of its events, taken from a registered message of that package.
func eventPackages() map[string]string {
	return map[string]string{
		"emissionstypes": protoPackageOf((*emissionstypes.EventAddStake)(nil)),
		"minttypes":      protoPackageOf((*minttypes.EventEcosystemTokenMintSet)(nil)),
	}
}

func protoPackageOf(msg proto.Message) string {
	name := proto.MessageName(msg)
	return name[:strings.LastIndex(name, ".")]
}

// notAuditedEvents lists the registered event types that have no rule in the
// audit, keyed by proto name, each with the reason. A type that gains a rule
// must leave this list.
var notAuditedEvents = buildNotAuditedEvents()

func buildNotAuditedEvents() map[string]string {
	out := make(map[string]string)
	exempt := func(reason string, events ...proto.Message) {
		for _, event := range events {
			out[proto.MessageName(event)] = reason
		}
	}
	exempt("whitelist membership or switch change: carries only the changed address or topic, with no relation to other events or ledgers to check",
		(*emissionstypes.EventGlobalAdminWhitelistAdded)(nil), (*emissionstypes.EventGlobalAdminWhitelistRemoved)(nil),
		(*emissionstypes.EventGlobalForecasterWhitelistAdded)(nil), (*emissionstypes.EventGlobalForecasterWhitelistRemoved)(nil),
		(*emissionstypes.EventGlobalReputerWhitelistAdded)(nil), (*emissionstypes.EventGlobalReputerWhitelistRemoved)(nil),
		(*emissionstypes.EventGlobalWhitelistAdded)(nil), (*emissionstypes.EventGlobalWhitelistRemoved)(nil),
		(*emissionstypes.EventGlobalWorkerWhitelistAdded)(nil), (*emissionstypes.EventGlobalWorkerWhitelistRemoved)(nil),
		(*emissionstypes.EventTopicCreatorWhitelistAdded)(nil), (*emissionstypes.EventTopicCreatorWhitelistRemoved)(nil),
		(*emissionstypes.EventTopicForecasterWhitelistAdded)(nil), (*emissionstypes.EventTopicForecasterWhitelistDisabled)(nil),
		(*emissionstypes.EventTopicForecasterWhitelistEnabled)(nil), (*emissionstypes.EventTopicForecasterWhitelistRemoved)(nil),
		(*emissionstypes.EventTopicReputerWhitelistAdded)(nil), (*emissionstypes.EventTopicReputerWhitelistDisabled)(nil),
		(*emissionstypes.EventTopicReputerWhitelistEnabled)(nil), (*emissionstypes.EventTopicReputerWhitelistRemoved)(nil),
		(*emissionstypes.EventTopicWorkerWhitelistAdded)(nil), (*emissionstypes.EventTopicWorkerWhitelistDisabled)(nil),
		(*emissionstypes.EventTopicWorkerWhitelistEnabled)(nil), (*emissionstypes.EventTopicWorkerWhitelistRemoved)(nil),
		(*emissionstypes.EventWhitelistAdminAdded)(nil), (*emissionstypes.EventWhitelistAdminRemoved)(nil),
	)
	exempt("actor registration change: carries only the registered or unregistered actor, with no relation to other events or ledgers to check",
		(*emissionstypes.EventWorkerRegistered)(nil), (*emissionstypes.EventWorkerUnregistered)(nil),
		(*emissionstypes.EventReputerRegistered)(nil), (*emissionstypes.EventReputerUnregistered)(nil),
		(*emissionstypes.EventNodeOwnerUpdated)(nil),
	)
	exempt("stake removal request bookkeeping: the stake ledger rule follows EventAddStake and EventRemoveStake, which move the balances",
		(*emissionstypes.EventRequestStakeRemoval)(nil), (*emissionstypes.EventCancelStakeRemoval)(nil),
	)
	exempt("derived from EventAddStake and EventRemoveStake, which the stake ledger rule already checks",
		(*emissionstypes.EventReputerStakeUpdated)(nil),
	)
	exempt("reward restake notice for a delegator: carries the amount restaked, with no relation to other events to check",
		(*emissionstypes.EventRewardDelegateStake)(nil),
	)
	exempt("parameter, topic or tokenomics settings change: carries the new values, which the audit does not cross-check against queried state",
		(*emissionstypes.EventParamsSet)(nil), (*emissionstypes.EventTopicUpdated)(nil),
		(*minttypes.EventParamsSet)(nil), (*minttypes.EventTokenomicsSet)(nil),
	)
	exempt("mint bookkeeping event: the audit's mint rules read EventEcosystemTokenMintSet and EventRewardCurrentBlockEmission",
		(*minttypes.EventEmissionInfo)(nil), (*minttypes.EventRecalculateTargetEmission)(nil),
	)
	exempt("pruning notice: carries only the height and topic, with no relation to other events or ledgers to check",
		(*emissionstypes.EventPruneRecords)(nil),
	)
	exempt("only the parallel-list length guard inspects it; no rule checks its values",
		(*emissionstypes.EventEMAScoresSet)(nil), (*emissionstypes.EventTopicRewardsSet)(nil),
	)
	exempt("used only to order topic status events within a block; no rule checks its payload",
		(*emissionstypes.EventFundTopic)(nil),
	)
	exempt("nested payload message embedded in other events, not emitted on its own",
		(*emissionstypes.EventValueBundle)(nil),
	)
	return out
}

// parseRuleFunctions parses the audit sources and returns the declaration of
// each function in auditRuleFunctions. A listed function that does not exist is
// missing from the result.
func parseRuleFunctions(t *testing.T) (found map[ruleFunction]*ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	found = make(map[ruleFunction]*ast.FuncDecl)
	files := make(map[string]*ast.File)
	for _, rule := range auditRuleFunctions {
		file, parsed := files[rule.file]
		if !parsed {
			var err error
			file, err = parser.ParseFile(fset, rule.file, nil, 0)
			require.NoError(t, err)
			files[rule.file] = file
		}
		for _, decl := range file.Decls {
			if fn, isFunc := decl.(*ast.FuncDecl); isFunc && fn.Body != nil && fn.Name.Name == rule.name {
				found[rule] = fn
			}
		}
	}
	return found
}

// Every function the scan relies on exists, so the list cannot go stale.
func TestAuditRuleFunctionsExist(t *testing.T) {
	found := parseRuleFunctions(t)
	for _, rule := range auditRuleFunctions {
		require.Contains(t, found, rule, "rule function %s is not declared in %s", rule.name, rule.file)
	}
}

// auditedEventTypes returns the proto names of the event types the rule
// functions check: every type a type-switch case or a type assertion in them
// names. The names come from the audit's own code, so the set cannot drift from
// what the audit does.
func auditedEventTypes(t *testing.T) map[string]bool {
	t.Helper()
	packages := eventPackages()
	audited := make(map[string]bool)
	record := func(expr ast.Expr) {
		star, ok := expr.(*ast.StarExpr)
		if !ok {
			return
		}
		selector, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			return
		}
		alias, ok := selector.X.(*ast.Ident)
		if !ok {
			return
		}
		if pkg, known := packages[alias.Name]; known && strings.HasPrefix(selector.Sel.Name, "Event") {
			audited[pkg+"."+selector.Sel.Name] = true
		}
	}
	for _, fn := range parseRuleFunctions(t) { //nolint:maprange // the result is a set, so order does not matter
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CaseClause:
				for _, expr := range n.List {
					record(expr)
				}
			case *ast.TypeAssertExpr:
				if n.Type != nil {
					record(n.Type)
				}
			}
			return true
		})
	}
	return audited
}

// registeredEventTypes returns every message named Event* in the proto
// packages of the audited modules, read from the proto registry.
func registeredEventTypes() map[string]bool {
	packages := make(map[string]bool)
	for _, pkg := range eventPackages() {
		packages[pkg] = true
	}
	registered := make(map[string]bool)
	proto.HybridResolver.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !packages[string(file.Package())] {
			return true
		}
		messages := file.Messages()
		for i := 0; i < messages.Len(); i++ {
			name := messages.Get(i).FullName()
			if strings.HasPrefix(string(name.Name()), "Event") {
				registered[string(name)] = true
			}
		}
		return true
	})
	return registered
}

// Every registered event type has a rule in the audit or an entry in
// notAuditedEvents. A new event fails here until one of the two exists.
func TestEveryRegisteredEventIsAuditedOrExempt(t *testing.T) {
	registered := registeredEventTypes()
	require.NotEmpty(t, registered, "no event types found in the proto registry")
	audited := auditedEventTypes(t)

	var unclassified, stale, contradictory, unknownAudited []string
	for name := range registered { //nolint:maprange // the lists are sorted before they are reported
		_, exempt := notAuditedEvents[name]
		switch {
		case audited[name] && exempt:
			contradictory = append(contradictory, name)
		case !audited[name] && !exempt:
			unclassified = append(unclassified, name)
		}
	}
	for name, reason := range notAuditedEvents { //nolint:maprange // the lists are sorted before they are reported
		if !registered[name] {
			stale = append(stale, name)
		}
		require.NotEmpty(t, strings.TrimSpace(reason), "%s has no reason", name)
	}
	for name := range audited { //nolint:maprange // the lists are sorted before they are reported
		if !registered[name] {
			unknownAudited = append(unknownAudited, name)
		}
	}
	for _, list := range [][]string{unclassified, stale, contradictory, unknownAudited} {
		sort.Strings(list)
	}

	require.Empty(t, unclassified, "event types with neither an audit rule nor a notAuditedEvents entry")
	require.Empty(t, contradictory, "event types with an audit rule that are also listed in notAuditedEvents")
	require.Empty(t, stale, "notAuditedEvents entries that are not registered event types")
	require.Empty(t, unknownAudited, "audit rules for types that are not registered event types")
}
