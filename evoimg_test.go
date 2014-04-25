package evoimage

import (
	"testing"
)

func TestTreeShake(t *testing.T) {
	// pares de expresiones con la original y el leído + shaken
	cases := []struct {
		a, b string
	}{
		{"[x|y|y]", "[rgb:x]"},
		{"[+ 1 2|x|y|r]", "[rgb:+ 1 2|x|y]"},
		{"[+ 1 3|x|r|y]", "[rgb:+ 1 2|x|y]"},
		{"[+ 2 3|r|x|y]", "[rgb:+ 1 2|x|y]"},
		{"[+ 2 4|r|x|r|y]", "[rgb:+ 1 2|x|y]"},
		{"[r:x|g:y|b:y]", "[r:x|g:y|b:y]"},
		{"[rg:x|x|b:y]", "[rg:x|b:y]"},
		{"[r:x|g:r|b:y]", "[r:x|g:r|b:y]"},
		{"[= 1|= 2|= 3]", "[rgb:= 1]"},
		{"[rgb:lerp 1 2 3|inv 2|x|band 4|y]", "[rgb:lerp 1 3 2|inv 3|band 4|x|y]"},
		{"[rgb:* 1 2|x|inv 1]", "[rgb:* 2 1|inv 2|x]"},
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

func TestEvalNodes(t *testing.T) {
	e, err := Read("[+ 1 2|x|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for x := 0.1; x < 1.0; x += .1 {
		if e.EvalNodes(x, .5, 1)[1] != x {
			t.Errorf("Node 1 in '%s' should eval to %g", e.String(), x)
		}
		if e.EvalNodes(0, x, 2)[2] != x {
			t.Errorf("Node 2 in '%s' should eval to %g", e.String(), x)
		}
	}
	e, err = Read("[blur 1 3|band 2|x|= 1]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for y := 0.1; y < 1.0; y += .1 {
		if v := e.EvalNodes(.5, y, 1); v[1] != 1.0 {
			t.Errorf("Node 1 in '%s' should eval to %g (evals to %g)", e.String(), 1.0, v)
		}
	}
	e, err = Read("[rgb:lerp 1 2 3|inv 2|x|band 4|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	e.EvalNodes(0, 0, 1)
}
