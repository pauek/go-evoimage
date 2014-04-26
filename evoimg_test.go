package evoimage

import (
	"testing"
)

func TestRead(t *testing.T) {
	cases := []struct{ a, b string }{
		{
			"(rgb)main(xy)[rgb:  x|y |y]",
			"(rgb)main(xy)[rgb:x|y|y]",
		}, {
			"(xyz)main(xyr)[xyz:  + 1  2 | x| y | r]",
			"(xyz)main(xyr)[xyz:+ 1 2|x|y|r]",
		}, {
			"(pqr)BLA(abc)[p:+ 1 3 | q:a|  r: b| c]",
			"(pqr)BLA(abc)[p:+ 1 3|q:a|r:b|c]",
		}, {
			"(mno)ASDF(pqr)[m:+ 2 3|no:p|q|r]",
			"(mno)ASDF(pqr)[m:+ 2 3|no:p|q|r]",
		}, {
			"(rgb)jarl(xry)[  + 2 4  | r:  r|  x| g:r|b:y]",
			"(rgb)jarl(xry)[+ 2 4|r:r|x|g:r|b:y]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:y]",
			"(rgb)(xy)[r:x|g:y|b:y]",
		}, {
			"(rgb)(xy)[rg:x|x|b:y]",
			"(rgb)(xy)[rg:x|x|b:y]",
		}, {
			"(rgb)(xry)[r:x|g:r|b:y]",
			"(rgb)(xry)[r:x|g:r|b:y]",
		}, {
			"(rgb)()[r:= 1|g:= 2|b:= 3]",
			"(rgb)()[r:= 1|g:= 2|b:= 3]",
		}, {
			"(rgb)___(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]",
			"(rgb)___(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]",
		}, {
			"(uvw)(xy)[uv:x|x|w:y]",
			"(uvw)(xy)[uv:x|x|w:y]",
		}, {
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
		},
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
	cases := []struct{ a, b string }{
		{
			"(rgb)(xy)[rgb:  x|y |y]",
			"(rgb)(x)[rgb:x]",
		}, {
			"(rbg)(xyr)[rgb:  + 1  2 | x| y | r]",
			"(rbg)(xy)[rbg:+ 1 2|x|y]",
		}, {
			"(bgr)(xyr)[r:+ 1 3 | g:x|  b: r| y]",
			"(bgr)(xyr)[b:r|g:x|r:+ 1 3|y]",
		}, {
			"(bgr)(xyr)[r:+ 1 3|b:r|g:x|y]",
			"(bgr)(xyr)[b:r|g:x|r:+ 0 3|y]",
		}, {
			"(ijk)(xy)[i:+ 2 3|= 1|jk:x|y]",
			"(ijk)(xy)[i:+ 1 2|jk:x|y]",
		}, {
			"(abc)(xy)[+ 2 4|x|ab:x|y|c:y]",
			"(abc)(xy)[ab:x|c:y]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:y]",
			"(rgb)(xy)[r:x|g:y|b:y]",
		}, {
			"(uvw)(xy)[uv:x|x|w:y]",
			"(uvw)(xy)[uv:x|w:y]",
		}, {
			"(rgb)()[rgb:= 1|= 2|= 3]",
			"(rgb)()[rgb:= 1]",
		}, {
			"(rgb)(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]",
			"(rgb)(xy)[rgb:lerp 1 3 2|inv 3|band 4|x|y]",
		}, {
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
			"(rgb)(x)[rgb:* 2 1|inv 2|x]",
		}, {
			"(p)(abc)[p:a|b|c]",
			"(p)(a)[p:a]",
		}, {
			"(p)(abc)[p:+ 2 1|b|c]",
			"(p)(bc)[p:+ 2 1|b|c]",
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
	e, err := Read("(a)(xy)[a:+ 1 2|x|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for x := 0.1; x < 1.0; x += .1 {
		e.SetInputs([]float64{x, .5})
		if e.EvalNodes(1); e.Nodes[1].Value != x {
			t.Errorf("Node 1 in '%s' should eval to %g", e.String(), x)
		}
		e.SetInputs([]float64{0, x})
		if e.EvalNodes(2); e.Nodes[2].Value != x {
			t.Errorf("Node 2 in '%s' should eval to %g", e.String(), x)
		}
	}
	e, err = Read("(y)(x)[y:+ 1 2|x|= 0.5]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for x := 0.1; x < 0.5; x += .05 {
		if out := e.Eval([]float64{x}); out[0] != ((x + 0.5) / 2.0) {
			t.Errorf("'%s' should eval to %g (evals to %g)",
				e.String(), ((x + 0.5) / 2.0), out[0])
		}
	}
	e, err = Read("(rgb)(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
}
