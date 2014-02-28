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