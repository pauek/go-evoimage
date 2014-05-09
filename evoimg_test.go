package evoimage

import (
	"math"
	"testing"
)

func TestReadErrorsModule(t *testing.T) {
	cases := []struct{ smod, serror string }{
		{
			"[x|y|y]",
			"Modules must have format `(abc)name(xyz)[...]`",
		}, {
			"(y)f(ab)[+ 1 2|a|b]",
			"Missing output `y`",
		}, {
			"", // empty string
			"Module is empty",
		}, {
			"()()[]", // empty expression
			"Empty node",
		}, {
			"(x)()[x:+ 1 2]", // missing nodes
			"Nonexistent node 1",
		}, {
			"(y)(x)[y:+ 1|x]", // wrong number of args
			"Error in node 0: `+` has 2 args, not 1.",
		}, {
			"(y)(x)[y:+ 1|x]", // wrong number of args
			"Error in node 0: `+` has 2 args, not 1.",
		},
	}
	for _, cas := range cases {
		_, err := readModule(cas.smod)
		if err == nil ||
			len(err.Error()) < len(cas.serror) ||
			err.Error()[:len(cas.serror)] != cas.serror {
			t.Errorf("Read should give '%s' error for '%s'", cas.serror, cas.smod)
			if err != nil {
				t.Logf("Error given is '%s'", err)
			} else {
				t.Log("No error given")
			}
		}
	}
}

func TestParseModule(t *testing.T) {
	cases := []struct{ a, b string }{
		{
			"(rgb)main(xy)[rgb:  x|y ]",
			"(rgb)main(xy)[rgb:x|y]",
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
			"(rgb)jarl(xryt)[  + 2 4  | r:  r|  x| g:t|b:y]",
			"(rgb)jarl(xryt)[+ 2 4|r:r|x|g:t|b:y]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:= 1]",
			"(rgb)(xy)[r:x|g:y|b:= 1]",
		}, {
			"(rgb)(xy)[rg:x|= 1|b:y]",
			"(rgb)(xy)[rg:x|= 1|b:y]",
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
			"(uvw)(xy)[uv:x|= 0|w:y]",
			"(uvw)(xy)[uv:x|= 0|w:y]",
		}, {
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
		},
	}

	// test read (no topological sort, no treeshake)
	for _, c := range cases {
		e1, err := parseModule(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		if s1 := e1.String(); s1 != c.b {
			t.Errorf("Error: reading '%s' gives '%s' (should be '%s')", c.a, s1, c.b)
		}
	}
}

func TestTopologicalSort(t *testing.T) {
	cases := []struct{ a, b string }{
		{
			"(rgb)(xy)[rgb:  x|y ]",
			"(rgb)(xy)[rgb:x|y]",
		}, {
			"(rbg)(xyr)[rgb:  + 1  2 | x| y | r]",
			"(rbg)(xyr)[rbg:+ 1 2|x|y|r]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:+ 0 1]",
			"(rgb)(xy)[b:+ 1 2|r:x|g:y]",
		}, {
			"(bgr)(xyr)[r:+ 1 3 | g:x|  b: r| y]",
			"(bgr)(xyr)[r:+ 1 3|g:x|b:r|y]",
		}, {
			"(bgr)(xyr)[r:+ 1 3|b:r|g:x|y]",
			"(bgr)(xyr)[r:+ 1 3|b:r|g:x|y]",
		}, {
			"(ijk)(xy)[= 1|i:+ 2 3|jk:x|y]",
			"(ijk)(xy)[i:+ 2 3|= 1|jk:x|y]",
		}, {
			"(abc)(xy)[= 0.2|+ 2 4|ab:x|= 0.3|c:y]",
			"(abc)(xy)[+ 2 4|= 0.2|ab:x|= 0.3|c:y]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:= 2]",
			"(rgb)(xy)[r:x|g:y|b:= 2]",
		}, {
			"(uvw)(xy)[uv:x|= 5|w:y]",
			"(uvw)(xy)[uv:x|= 5|w:y]",
		}, {
			"(rgb)()[rgb:= 1|= 2|= 3]",
			"(rgb)()[rgb:= 1|= 2|= 3]",
		}, {
			"(rgb)(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]",
			"(rgb)(xy)[rgb:lerp 1 3 2|inv 3|band 4|x|y]",
		}, {
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
			"(rgb)(x)[rgb:* 2 1|inv 2|x]",
		},
	}
	// test topological sort only
	for _, c := range cases {
		e1, err := parseModule(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		e1.TopologicalSort()
		if s1 := e1.String(); s1 != c.b {
			t.Errorf("Error: topological sort of '%s' gives '%s' (should be '%s')", c.a, s1, c.b)
		}
	}
}

func TestSortAndTreeShake(t *testing.T) {
	cases := []struct{ a, b string }{
		{
			"(rgb)A1(xy)[rgb:  x|y ]",
			"(rgb)A1(xy)[rgb:x]",
		}, {
			"(rbg)A2(xyr)[rgb:  + 1  2 | x| y | r]",
			"(rbg)A2(xyr)[rbg:+ 1 2|x|y]",
		}, {
			"(bgr)pauek(xyr)[r:+ 1 3 | g:x|  b: r| y |bla]",
			"(bgr)pauek(xyr)[r:+ 1 3|g:x|b:r|y]",
		}, {
			"(bgr)(xyr)[r:+ 1 3|b:r|g:x|y]",
			"(bgr)(xyr)[r:+ 1 3|b:r|g:x|y]",
		}, {
			"(ijk)(xy)[i:+ 2 3|= 1|jk:x|y]",
			"(ijk)(xy)[i:+ 1 2|jk:x|y]",
		}, {
			"(abc)(xy)[+ 2 4|= 0|ab:x|= 1|c:y]",
			"(abc)(xy)[ab:x|c:y]",
		}, {
			"(abc)(xy)[a:+ 2 4|= 0.5|b:x|= 0.2|c:y]",
			"(abc)(xy)[a:+ 1 2|b:x|c:y]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:= 1]",
			"(rgb)(xy)[r:x|g:y|b:= 1]",
		}, {
			"(rgb)(xy)[r:x|g:y|b:+ 0 1]",
			"(rgb)(xy)[b:+ 1 2|r:x|g:y]",
		}, {
			"(uvw)(xyr)[uv:x|= 1|w:y]",
			"(uvw)(xyr)[uv:x|w:y]",
		}, {
			"(rgb)(x)[rgb:= 1|= 2|= 3]",
			"(rgb)(x)[rgb:= 1]",
		}, {
			"(rgb)(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]",
			"(rgb)(xy)[rgb:lerp 1 3 2|inv 3|band 4|x|y]",
		}, {
			"(rgb)(x)[rgb:* 1 2|x|inv 1]",
			"(rgb)(x)[rgb:* 2 1|inv 2|x]",
		}, {
			"(p)(abc)[p:a|b|c]",
			"(p)(abc)[p:a]",
		}, {
			"(p)(abc)[p:+ 2 1|b|c]",
			"(p)(abc)[p:+ 2 1|b|c]",
		},
	}

	// test read (no topological sort, no treeshake)
	for _, c := range cases {
		e1, err := readModule(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		if s1 := e1.String(); s1 != c.b {
			t.Errorf("Error: sorting + shaking '%s' gives '%s' (should be '%s')", c.a, s1, c.b)
		}
	}
}

func TestReadModuleName(t *testing.T) {
	cases := []struct{ a, b string }{
		{
			"(rgb)asdf(xy)[rgb:x|y|y]",
			"asdf",
		},
	}
	for _, c := range cases {
		m, _ := readModule(c.a)
		if m.Name != c.b {
			t.Errorf("Module '%s' should have name '%s' (has '%s')", c.a, c.b, m.Name)
		}
	}
}

func TestEvalNodes(t *testing.T) {
	e, err := readModule("(a)(xy)[a:+ 1 2|x|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for x := 0.1; x < 1.0; x += .1 {
		e.SetInputs([]float64{x, .5})
		if e.EvalNodes(nil, 1); e.Nodes[1].Value[0] != x {
			t.Errorf("Node 1 in '%s' should eval to %g", e.String(), x)
		}
		e.SetInputs([]float64{0, x})
		if e.EvalNodes(nil, 2); e.Nodes[2].Value[0] != x {
			t.Errorf("Node 2 in '%s' should eval to %g", e.String(), x)
		}
	}
	e, err = readModule("(y)(x)[y:+ 1 2|x|= 0.5]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
	for x := 0.1; x < 0.5; x += .05 {
		if out := e.Eval(nil, []float64{x}); out[0] != ((x + 0.5) / 2.0) {
			t.Errorf("'%s' should eval to %g (evals to %g)",
				e.String(), ((x + 0.5) / 2.0), out[0])
		}
	}
	e, err = readModule("(rgb)(xy)[rgb:lerp 1 2 3|inv 2|x|band 4|y]")
	if err != nil {
		t.Errorf("Error reading expression: %s", err)
	}
}

func TestReadErrorsCircuit(t *testing.T) {
	cases := []struct{ smod, serror string }{
		{
			"(rgb)asdf(xy)[rgb:x|y]",
			"There is no main module",
		}, {
			"(rgb)asdf(xy)[rgb:x|y|y]",
			"Duplicated input 'y'",
		}, {
			"(r)(x)[r:x]",
			"Outputs != 'rgb'!",
		}, {
			"(abc)(x)[abc:x]",
			"Outputs != 'rgb'!",
		}, {
			"(rgb)(xyrt)[r:+ 1 2|g:+ 3 4|b:sum 5 6|x|y|r|t]",
			"Missing module `sum`",
		}, {
			"(rgb)(xyrt)[r:+ 1 2|g:+ 3 4|b:sum 5 6|x|y|r|t];(fg)sum(xy)[f:+ 1 2|g:x|y]",
			"Module `sum` has more than one output",
		}, {
			"(rgb)(xy)[rgb:sum 1 2|x|y];(f)sum(xyz)[f:+ 1 2|x|+ 3 4|y|z]",
			"Module `sum` has 3 inputs, not 2.",
		}, {
			"(rgb)(x)[rgb:x];(y)a(x)[y:x];(w)a(v)[w:v]",
			"Duplicated module `a`.",
		},
	}
	for _, cas := range cases {
		C, err := Read(cas.smod)
		if err == nil ||
			len(err.Error()) < len(cas.serror) ||
			err.Error()[:len(cas.serror)] != cas.serror {
			t.Errorf("Read should give '%s' error for '%s'", cas.serror, cas.smod)
			t.Log(C)
			if err != nil {
				t.Logf("Error given is '%s'", err)
			} else {
				t.Log("No error given")
			}
		}
	}
}

func TestCircuitEval(t *testing.T) {
	cases := []struct {
		circuit string
		inputs  []float64
		outputs []float64
	}{
		{
			"(rgb)(xy)[r:x|gb:y]",
			[]float64{0.1, 0.9 /* two extra inputs */, 0.0, 0.0},
			[]float64{0.1, 0.9, 0.9},
		}, {
			"(rgb)(xy)[b:+ 1 2|r:x|g:y]",
			[]float64{0.2, 0.4},
			[]float64{0.2, 0.4, 0.3},
		}, {
			"(rgb)(x)[rgb:mod1 1|x];(x)mod1(y)[x:y]",
			[]float64{0.5},
			[]float64{0.5, 0.5, 0.5},
		}, {
			"(rgb)(xy)[r:mult 1 2|g:x|b:y];(f)mult(xy)[f:* 1 2|x|y]",
			[]float64{0.5, 0.3},
			[]float64{0.15, 0.5, 0.3},
		},
	}
	for _, cas := range cases {
		C, err := Read(cas.circuit)
		if err != nil {
			t.Errorf("Cannot read '%s': %s", cas.circuit, err)
		}
		outputs := C.Eval(cas.inputs)
		if len(outputs) != len(cas.outputs) {
			t.Errorf("Different number of outputs for '%s': %#v versus %#v",
				cas.circuit, outputs, cas.outputs)
		} else {
			for i := range outputs {
				if math.Abs(outputs[i]-cas.outputs[i]) > 1e-9 {
					t.Errorf("Different value for output %d: %f vs. %f", i, outputs[i], cas.outputs[i])
					t.Logf("Difference = %f", outputs[i]-cas.outputs[i])
				}
			}
		}
	}
}
