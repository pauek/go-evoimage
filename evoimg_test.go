package evoimage

import (
	"testing"
)

func TestRead(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"[rgb:  x|y |y]", "[rgb:x|y|y]"},
		{"[xyz:  + 1  2 | x| y | r]", "[xyz:+ 1 2|x|y|r]"},
		{"[r:+ 1 3 | g:x|  b: r| y]", "[r:+ 1 3|g:x|b:r|y]"},
		{"[r:+ 2 3|gb:r|x|y]", "[r:+ 2 3|gb:r|x|y]"},
		{"[+ 2 4|r:r|x|g:r|b:y]", "[+ 2 4|r:r|x|g:r|b:y]"},
		{"[r:x|g:y|b:y]", "[r:x|g:y|b:y]"},
		{"[rg:x|x|b:y]", "[rg:x|x|b:y]"},
		{"[r:x|g:r|b:y]", "[r:x|g:r|b:y]"},
		{"[r:= 1|g:= 2|b:= 3]", "[r:= 1|g:= 2|b:= 3]"},
		{"[rgb:lerp 1 2 3|inv 2|x|band 4|y]", "[rgb:lerp 1 2 3|inv 2|x|band 4|y]"},
		{"[rgb:* 1 2|x|inv 1]", "[rgb:* 1 2|x|inv 1]"},
	}

	// test read (no topological sort, no treeshake)
	for _, c := range cases {
		e1, err := read(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		if s1 := e1.String(); s1 != c.b {
			t.Errorf("Error: reading '%s' gives '%s' (should be '%s')", c.a, s1, c.b)
		}
	}

	// missing outputs
	no_outputs := "[x|y|y]"
	_, err := read(no_outputs)
	if err == nil {
		t.Errorf("Reading should give an error and it doesn't for '%s'", no_outputs)
	}

	// empty expression
	empty := []string{"[]", ""}
	for _, e := range empty {
		_, err := read(e)
		if err == nil {
			t.Errorf("Reading should give an error and it doesn't for '%s'", e)
		}
	}
}

func TestSortAndTreeShake(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"[rgb:  x|y |y]", "[rgb:x]"},
		{"[rgb:  + 1  2 | x| y | r]", "[rgb:+ 1 2|x|y]"},
		{"[r:+ 1 3 | g:x|  b: r| y]", "[r:+ 1 3|g:x|b:r|y]"},
		{"[r:+ 2 3|r|gb:x|y]", "[r:+ 1 2|gb:x|y]"},
		{"[+ 2 4|r|ab:x|r|c:y]", "[ab:x|c:y]"},
		{"[r:x|g:y|b:y]", "[r:x|g:y|b:y]"},
		{"[rg:x|x|b:y]", "[rg:x|b:y]"},
		{
			"[rgb:= 1|= 2|= 3]",
			"[rgb:= 1]",
		},
		{
			"[rgb:lerp 1 2 3|inv 2|x|band 4|y]",
			"[rgb:lerp 1 3 2|inv 3|band 4|x|y]",
		},
		{
			"[rgb:* 1 2|x|inv 1]",
			"[rgb:* 2 1|inv 2|x]",
		},
	}

	// test read (no topological sort, no treeshake)
	for _, c := range cases {
		e1, err := Read(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		if s1 := e1.String(); s1 != c.b {
			t.Errorf("Error: sorting + shaking '%s' gives '%s' (should be '%s')", c.a, s1, c.b)
		}
	}

}

func TestEvalNodes(t *testing.T) {
	e, err := Read("[a:+ 1 2|x|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for x := 0.1; x < 1.0; x += .1 {
		if e.EvalNodes(x, .5, 1); e.Nodes[1].Value != x {
			t.Errorf("Node 1 in '%s' should eval to %g", e.String(), x)
		}
		if e.EvalNodes(0, x, 2); e.Nodes[2].Value != x {
			t.Errorf("Node 2 in '%s' should eval to %g", e.String(), x)
		}
	}
	e, err = Read("[rgb:blur 1 3|band 2|x|= 1]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for y := 0.1; y < 1.0; y += .1 {
		if e.EvalNodes(.5, y, 1); e.Nodes[1].Value != 1.0 {
			t.Errorf("Node 1 in '%s' should eval to %g (evals to %g)",
				e.String(), 1.0, e.Nodes[1].Value)
		}
	}
	e, err = Read("[rgb:lerp 1 2 3|inv 2|x|band 4|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
}
