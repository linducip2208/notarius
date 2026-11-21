// Package plan implements office subscription tiers and feature gating.
// LITE: operasional inti. PRO: + portal bayar, share berjangka, checklist.
// ENTERPRISE: + lintas-kantor (acting), prioritas.
package plan

// Rank orders tiers.
func Rank(plan string) int {
	switch plan {
	case "ENTERPRISE":
		return 2
	case "PRO":
		return 1
	default:
		return 0 // LITE & unknown fail closed to lite
	}
}

// Feature minimum tiers.
const (
	FeatPortalPay = "portal.pay"
	FeatShares    = "doc.shares"
	FeatActing    = "office.acting"
	FeatChecklist = "doc.checklist"
)

func minRank(feature string) int {
	switch feature {
	case FeatPortalPay, FeatShares, FeatChecklist:
		return 1 // PRO
	case FeatActing:
		return 2 // ENTERPRISE
	default:
		return 0
	}
}

// Allowed reports whether plan covers feature.
func Allowed(plan, feature string) bool {
	return Rank(plan) >= minRank(feature)
}
