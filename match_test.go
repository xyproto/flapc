package main

import (
	"bytes"
	"testing"
)

func TestMatchNumericLiteralEmitsEqualityComparison(t *testing.T) {
	platform := Platform{Arch: ArchX86_64, OS: OSLinux}
	fc, err := NewFlapCompiler(platform)
	if err != nil {
		t.Fatalf("NewFlapCompiler: %v", err)
	}

	fc.eb.text.Reset()

	expr := &MatchExpr{
		Condition: &NumberExpr{Value: 2},
		Clauses: []*MatchClause{
			{
				Guard:  &NumberExpr{Value: 1},
				Result: &NumberExpr{Value: 10},
			},
			{
				Guard:  &NumberExpr{Value: 2},
				Result: &NumberExpr{Value: 20},
			},
		},
		DefaultExpr:     &NumberExpr{Value: 30},
		DefaultExplicit: true,
	}

	fc.compileMatchExpr(expr)

	code := fc.eb.text.Bytes()

	// Look for a ucomisd instruction that is followed by a JNE (0x0F 0x85)
	pattern := []byte{0x0F, 0x2E} // ucomisd opcode prefix
	idx := bytes.Index(code, pattern)
	found := false
	for idx != -1 {
		start := idx + len(pattern)
		end := start + 16
		if end > len(code) {
			end = len(code)
		}
		if bytes.Index(code[start:end], []byte{0x0F, 0x85}) != -1 {
			found = true
			break
		}
		next := bytes.Index(code[start:], pattern)
		if next == -1 {
			break
		}
		idx = start + next
	}

	if !found {
		t.Fatalf("expected numeric match guard to emit JNE after ucomisd; code=% x", code)
	}
}
