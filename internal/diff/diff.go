// Package diff implements line-based document comparison (LCS) for deed revisions.
// Output is a list of operations the UI renders as red/green highlighted lines.
package diff

// Op is one diff operation.
type Op struct {
	Kind string // " " same, "-" removed, "+" added
	Line string
}

// Lines splits text into lines (normalizes CRLF).
// Empty text yields no lines; a single trailing newline is not a line.
func Lines(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			ln := s[start:i]
			if len(ln) > 0 && ln[len(ln)-1] == '\r' {
				ln = ln[:len(ln)-1]
			}
			out = append(out, ln)
			start = i + 1
		}
	}
	if rest := s[start:]; rest != "" {
		out = append(out, rest)
	}
	return out
}

// Compare returns the LCS-based line diff old -> new.
// Complexity O(n*m) with cap: inputs truncated to 2000 lines each (honest limit for UI).
func Compare(oldText, newText string) []Op {
	a, b := Lines(oldText), Lines(newText)
	if len(a) > 2000 {
		a = a[:2000]
	}
	if len(b) > 2000 {
		b = b[:2000]
	}
	n, m := len(a), len(b)
	// LCS table (uint16 enough for 2000).
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var out []Op
	for i, j := 0, 0; i < n || j < m; {
		switch {
		case i < n && j < m && a[i] == b[j]:
			out = append(out, Op{" ", a[i]})
			i++
			j++
		case j < m && (i >= n || lcs[i][j+1] >= lcs[i+1][j]):
			out = append(out, Op{"+", b[j]})
			j++
		default:
			out = append(out, Op{"-", a[i]})
			i++
		}
	}
	return out
}

// Stats counts added/removed lines (for badges).
func Stats(ops []Op) (add, del int) {
	for _, o := range ops {
		switch o.Kind {
		case "+":
			add++
		case "-":
			del++
		}
	}
	return add, del
}
