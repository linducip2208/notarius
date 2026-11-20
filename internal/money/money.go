// Package money provides decimal-safe IDR monetary calculations.
// IDR has no minor units; all amounts are whole rupiah (int64).
package money

import (
	"errors"
	"fmt"
	"strings"
)

// Parse accepts "1500000", "1.500.000", "Rp 1.500.000" and returns rupiah.
func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "Rp")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, errors.New("amount is empty")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid amount %q", s)
		}
		n = n*10 + int64(r-'0')
		if n > 9_000_000_000_000_000 {
			return 0, errors.New("amount out of range")
		}
	}
	if neg {
		n = -n
	}
	return n, nil
}

func Add(a, b int64) int64 { return a + b }
func Sub(a, b int64) int64 { return a - b }

// Mul multiplies amount by qty, guarding overflow.
func Mul(amount int64, qty int) (int64, error) {
	if qty < 0 {
		return 0, errors.New("qty must be >= 0")
	}
	if amount != 0 && int64(qty) > 9_000_000_000_000_000/amount {
		return 0, errors.New("multiplication overflow")
	}
	return amount * int64(qty), nil
}

// PPN adds Indonesian VAT at rateBps basis points (e.g. 1100 = 11%).
func PPN(net int64, rateBps int) int64 { return net + net*int64(rateBps)/10000 }

// FormatIDR renders "Rp 1.500.000".
func FormatIDR(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if neg {
		return "Rp -" + b.String()
	}
	return "Rp " + b.String()
}
