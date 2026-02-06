package x

import (
	"math"
	"sort"
	"testing"
)

// --- Minimal helpers (standalone “proof” scaffolding) ---

// crossEntropyOneHot returns -ln(p_gt). If p_gt==0 -> +Inf.
// (This matches the “if GT label missing => prob 0 => infinite loss” semantics.)
func crossEntropyOneHot(pGT float64) float64 {
	if pGT <= 0 {
		return math.Inf(1)
	}
	return -math.Log(pGT)
}

// normalizations people usually propose
type normKind string

const (
	NormNone  normKind = "none"
	NormByN   normKind = "byN"   // loss / N
	NormByLog normKind = "byLog" // loss / ln(N)
)

func normalize(loss float64, n int, k normKind) float64 {
	switch k {
	case NormNone:
		return loss
	case NormByN:
		return loss / float64(n)
	case NormByLog:
		return loss / math.Log(float64(n))
	default:
		panic("unknown norm")
	}
}

// unionLabels returns the deterministic union of labels appearing in submissions.
func unionLabels(subs ...map[string]float64) []string {
	set := map[string]struct{}{}
	for _, s := range subs {
		for lab := range s {
			set[lab] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for lab := range set {
		out = append(out, lab)
	}
	sort.Strings(out)
	return out
}

// registryAfterAppendOnly simulates a topic-lifetime registry (append-only).
func registryAfterAppendOnly(initial []string, seenThisEpoch ...map[string]float64) []string {
	set := map[string]struct{}{}
	for _, l := range initial {
		set[l] = struct{}{}
	}
	// append-only: add any new labels ever seen
	for _, s := range seenThisEpoch {
		for lab := range s {
			set[lab] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for lab := range set {
		out = append(out, lab)
	}
	sort.Strings(out)
	return out
}

// uniformPredictor: assigns uniform prob across its label space (sum=1).
func uniformPgt(labelSpace []string, gt string) float64 {
	if len(labelSpace) == 0 {
		return 0
	}
	// if GT not in the label space, prob=0
	for _, l := range labelSpace {
		if l == gt {
			return 1.0 / float64(len(labelSpace))
		}
	}
	return 0
}

// skilledPredictor: assigns pGT to GT and spreads rest uniformly among remaining labels (sum=1).
func skilledPgt(labelSpace []string, gt string, pGT float64) float64 {
	// if GT not in label space, prob=0
	found := false
	for _, l := range labelSpace {
		if l == gt {
			found = true
			break
		}
	}
	if !found || len(labelSpace) == 0 {
		return 0
	}
	return pGT
}

// --- Tests proving the claims ---

func TestLossMeaningChangesWithLabelCount_EvenIfRawLossIsSame(t *testing.T) {
	// Same p(GT)=0.1 => same raw CE loss in both epochs,
	// but relative to the “random guess” baseline it flips from “worse than random” to “better than random”.
	type tc struct {
		name string
		n    int
		pGT  float64
	}
	cases := []tc{
		{name: "epoch_small_label_space", n: 3, pGT: 0.1},
		{name: "epoch_large_label_space", n: 30, pGT: 0.1},
	}

	rawLoss := crossEntropyOneHot(0.1)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := crossEntropyOneHot(c.pGT)
			if math.Abs(got-rawLoss) > 1e-12 {
				t.Fatalf("expected same raw loss, got=%v want=%v", got, rawLoss)
			}

			randomBaseline := math.Log(float64(c.n)) // uniform => pGT=1/n => loss=ln(n)
			isBetterThanRandom := got < randomBaseline
			// For n=3: ln(3)=1.098...; 2.302 > 1.098 => NOT better than random
			// For n=30: ln(30)=3.401...; 2.302 < 3.401 => better than random
			if c.n == 3 && isBetterThanRandom {
				t.Fatalf("should be worse than random when n=3")
			}
			if c.n == 30 && !isBetterThanRandom {
				t.Fatalf("should be better than random when n=30")
			}
		})
	}
}

func TestEpochUnionVsTopicRegistry_PollutionPersistsOnlyInTopicRegistry(t *testing.T) {
	// Setup:
	// - Epoch1: labels {A,B,C}
	// - Epoch2: attacker submits many junk labels once
	// - Epoch3: only {A,B,C} are “active” again
	//
	// Epoch-union model:
	//   label space in epoch3 = union of labels present in epoch3 submissions => {A,B,C}
	//
	// Topic-lifetime registry:
	//   label space in epoch3 = registry that still contains junk from epoch2 => {A,B,C}+junk
	//
	// Show: a uniform predictor (uninformed worker) suffers forever under topic registry,
	// but not under epoch-union, even when junk labels are no longer used.
	initial := []string{"A", "B", "C"}

	epoch3Subs := []map[string]float64{
		{"A": 1, "B": 1, "C": 1}, // content doesn't matter; just indicates which labels are present
	}

	// epoch2 had pollution
	junk := map[string]float64{}
	for i := 0; i < 27; i++ {
		junk[string(rune('D'+i))] = 1
	}

	epoch3Union := unionLabels(epoch3Subs...)                            // {A,B,C}
	topicRegistry := registryAfterAppendOnly(initial, junk /* epoch2 */) // {A,B,C}+junk

	gt := "A"
	pUnion := uniformPgt(epoch3Union, gt)   // 1/3
	pTopic := uniformPgt(topicRegistry, gt) // 1/30

	lUnion := crossEntropyOneHot(pUnion) // ln(3)
	lTopic := crossEntropyOneHot(pTopic) // ln(30)

	if !(lTopic > lUnion) {
		t.Fatalf("expected topic-registry loss to be worse after pollution; union=%v topic=%v", lUnion, lTopic)
	}
}

func TestNormalizationDoesNotMakeThingsEquivalent_AndCanChangeRelativeSignal(t *testing.T) {
	// In epoch3 (after pollution), compare:
	// - a uniform worker
	// - a skilled worker (pGT=0.9)
	//
	// Under epoch-union: N=3
	// Under topic-registry: N=30 (junk persists)
	//
	// Show:
	// 1) raw losses differ across the two models
	// 2) “normalize by N” still differs
	// 3) “normalize by log(N)” equalizes the uniform baseline, BUT it changes the scale of the skilled signal
	//    (i.e., the same skilled behavior looks *more* impressive when N is larger), which distorts regret dynamics.
	unionN := 3
	regN := 30

	unionSpace := []string{"A", "B", "C"}
	regSpace := make([]string, 0, regN)
	regSpace = append(regSpace, "A", "B", "C")
	for i := 0; i < 27; i++ {
		regSpace = append(regSpace, string(rune('D'+i)))
	}

	gt := "A"
	pSkilled := 0.9

	// Uniform
	lU_union := crossEntropyOneHot(uniformPgt(unionSpace, gt)) // ln(3)
	lU_reg := crossEntropyOneHot(uniformPgt(regSpace, gt))     // ln(30)

	// Skilled (same pGT in both models)
	lS_union := crossEntropyOneHot(skilledPgt(unionSpace, gt, pSkilled)) // -ln(0.9)
	lS_reg := crossEntropyOneHot(skilledPgt(regSpace, gt, pSkilled))     // -ln(0.9) (same raw)

	if math.Abs(lS_union-lS_reg) > 1e-12 {
		t.Fatalf("expected skilled raw loss to match across models; got union=%v reg=%v", lS_union, lS_reg)
	}
	if !(lU_reg > lU_union) {
		t.Fatalf("expected uniform raw loss to be worse with larger registry; union=%v reg=%v", lU_union, lU_reg)
	}

	// Check normalizations.
	norms := []normKind{NormByN, NormByLog}
	for _, nk := range norms {
		t.Run(string(nk), func(t *testing.T) {
			uUnion := normalize(lU_union, unionN, nk)
			uReg := normalize(lU_reg, regN, nk)
			sUnion := normalize(lS_union, unionN, nk)
			sReg := normalize(lS_reg, regN, nk)

			switch nk {
			case NormByN:
				// Still not equivalent: different numbers.
				if math.Abs(uUnion-uReg) < 1e-9 {
					t.Fatalf("did not expect by-N to equalize uniform across models")
				}
			case NormByLog:
				// Uniform becomes exactly 1 in both (since ln(N)/ln(N)=1)
				if math.Abs(uUnion-1.0) > 1e-9 || math.Abs(uReg-1.0) > 1e-9 {
					t.Fatalf("expected uniform to normalize to 1 by-log; got union=%v reg=%v", uUnion, uReg)
				}
				// But skilled “signal” changes: same skilled behavior yields different normalized loss.
				// (Because we divide by ln(N): larger N makes the same raw loss look smaller.)
				if !(sReg < sUnion) {
					t.Fatalf("expected skilled normalized loss to be smaller when N is larger; union=%v reg=%v", sUnion, sReg)
				}
			}

			// Also show the *gap* changes (this is what regret/weights would feel).
			gapUnion := uUnion - sUnion
			gapReg := uReg - sReg
			if math.Abs(gapUnion-gapReg) < 1e-9 {
				t.Fatalf("expected the gap (signal strength) to differ across models; unionGap=%v regGap=%v", gapUnion, gapReg)
			}
		})
	}
}
