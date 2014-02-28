package evoimage

import (
	"testing"
)

func TestTreeShake(t *testing.T) {
	// pares de expresiones con la original y el shaken
	s012 := []int{0, 1, 2}
	s0 := []int{0}
	cases := []struct { 
		a, b string 
		shk []int
	}{
		{"[x; y; y]", "[x; y; y]", s012},
		{"[+ 1 2; x; y; r]", "[+ 1 2; x; y]", s0},
		{"[+ 1 3; x; r; y]", "[+ 1 2; x; y]", s0},
		{"[+ 2 3; r; x; y]", "[+ 1 2; x; y]", s0},
		{"[+ 2 4; r; x; r; y]", "[+ 1 2; x; y]", s0},
		{"[= 1; = 2; = 3]", "[= 1; = 2; = 3]", s012},
		{"[= 1; = 2; = 3]", "[= 1]", s0},
	}
	for _, c := range cases {
		e1, err := Read(c.a)
		if err != nil {
			t.Errorf("Cannot read expression '%s': %s", c, err)
		}
		s1 := e1.String()
		s2 := e1.TreeShake(c.shk...).String()
		if s2 != c.b {
			t.Errorf("Error: shaking '%s' gives '%s' (should be '%s')", s1, s2, c.b)
		}
	}
}