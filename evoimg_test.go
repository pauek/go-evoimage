package evoimage

import (
	"testing"
)

func TestTreeShake(t *testing.T) {
	// pares de expresiones con la original y el shaken
	cases := []struct { 
		a, b string 
	}{
		{"[x; y; y]", "[x]"},
		{"[+ 1 2; x; y; r]", "[+ 1 2; x; y]"},
		{"[+ 1 3; x; r; y]", "[+ 1 2; x; y]"},
		{"[+ 2 3; r; x; y]", "[+ 1 2; x; y]"},
		{"[+ 2 4; r; x; r; y]", "[+ 1 2; x; y]"},
		{"[= 1; = 2; = 3]", "[= 1]"},
	}
	for _, c := range cases {
		e1, err := Read(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		if s1 := e1.String(); s1 != c.b {
			t.Errorf("Error: shaking '%s' gives '%s' (should be '%s')", c.a, s1, c.b)
		}
	}
}

func TestEvalNode(t *testing.T) {
	e, _ := Read("[+ 1 2; x; y]")
	for x := 0.1; x < 1.0; x += .1 {
		if e.EvalNode(1, x, .5) != x {
			t.Errorf("Node 1 in '%s' should eval to %g", e.String(), x)
		}
		if e.EvalNode(2, 0, x) != x {
			t.Errorf("Node 2 in '%s' should eval to %g", e.String(), x)
		}
	}
	e, _ = Read("[blur 1; band 2; x]")
	for y := 0.1; y < 1.0; y += .1 {
		if v := e.EvalNode(1, .5, y); v != 1.0 {
			t.Errorf("Node 1 in '%s' should eval to %g (evals to %g)", e.String(), 1.0, v)
		}
	}
}