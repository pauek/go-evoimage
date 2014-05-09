package evoimage

import (
	"fmt"
	"go-evoimage/perlin"
	"image"
	"image/color"
	"io"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"
)

var pnoise = perlin.NewPerlinNoise(time.Now().UnixNano())

func find(v int, seq []int) int {
	for i, x := range seq {
		if v == x {
			return i
		}
	}
	return -1
}

func uniq(seq []int) (res []int) {
	for _, x := range seq {
		if find(x, res) == -1 {
			res = append(res, x)
		}
	}
	return
}

const MAX_ARGS = 10

type OpInfo struct {
	Nargs int
}

var NumArguments = map[int][]string{
	0: {"="},
	1: {"x2", "x3", "cos", "sin", "tri", "inv", "band", "bw"},
	2: {"+", "*", "/", "-", "min", "max", "and", "or", "xor", "noise"},
	3: {"lerp", "if"},
}

var Operators = []string{}
var OperatorInfo = make(map[string]OpInfo)

func init() {
	for nargs, ops := range NumArguments {
		for _, op := range ops {
			Operators = append(Operators, op)
			OperatorInfo[op] = OpInfo{
				Nargs: nargs,
			}
		}
	}
}

type Color struct {
	R, G, B float64
}
type Argument int
type Node struct {
	Op    string
	Args  []Argument
	Value []float64
	Ready bool
	Call  bool
}
type _Node struct {
	Node
	Order  int
	NewPos int
}
type Port struct {
	Name rune
	Idx  int
}
type Module struct {
	Name    string
	Nodes   []*Node
	Inputs  []Port
	Outputs []Port
}
type Circuit struct {
	Modules map[string]*Module
}

func argument(node, output int) Argument {
	return Argument(node*10 + output%10)
}

var Unset = Argument(-1)

func (A Argument) IsUnset() bool { return int(A) == -1 }
func (A Argument) Node() int     { return int(A / 10) }
func (A Argument) Output() int   { return int(A % 10) }

func (c *Color) Add(other Color) {
	c.R += other.R
	c.G += other.G
	c.B += other.B
}

func (c *Color) Divide(x float64) Color {
	return Color{c.R / x, c.G / x, c.B / x}
}

func (N *Node) Clone() (node *Node) {
	node = &Node{
		Op:    N.Op,
		Value: make([]float64, len(N.Value)),
		Args:  make([]Argument, len(N.Args)),
	}
	copy(node.Value, N.Value)
	copy(node.Args, N.Args)
	return
}

func (node *Node) eval(M Module) {
	if node.Ready {
		return
	}
	switch node.Op {
	case "=":
		// Value is already there

	case "x2":
		i0 := node.Args[0]
		f := M.Nodes[i0.Node()].Value[i0.Output()]
		if f < .5 {
			node.Value[0] = 2.0 * f
		} else {
			node.Value[0] = 2.0*f - 1
		}

	case "x3":
		i0 := node.Args[0]
		f := M.Nodes[i0.Node()].Value[i0.Output()]
		if f < .3333 {
			node.Value[0] = 3.0 * f
		} else if f < .6666 {
			node.Value[0] = 3.0*f - 1
		} else {
			node.Value[0] = 3.0*f - 2
		}

	case "band":
		i0 := node.Args[0]
		a := M.Nodes[i0.Node()].Value[i0.Output()]
		if a > .33 && a < .66 {
			node.Value[0] = 1.0
		} else {
			node.Value[0] = 0.0
		}

	case "bw":
		i0 := node.Args[0]
		a := M.Nodes[i0.Node()].Value[i0.Output()]
		if a > .5 {
			node.Value[0] = 1.0
		} else {
			node.Value[0] = 0.0
		}

	case "inv":
		i0 := node.Args[0]
		a := M.Nodes[i0.Node()].Value[i0.Output()]
		node.Value[0] = (1 - a)

	case "cos":
		i0 := node.Args[0]
		f := M.Nodes[i0.Node()].Value[i0.Output()]
		node.Value[0] = (1 + math.Cos(2*math.Pi*f)) / 2

	case "sin":
		i0 := node.Args[0]
		f := M.Nodes[i0.Node()].Value[i0.Output()]
		node.Value[0] = (1 + math.Sin(2*math.Pi*f)) / 2

	case "tri":
		i0 := node.Args[0]
		f := M.Nodes[i0.Node()].Value[i0.Output()]
		if f < .5 {
			node.Value[0] = 2.0 * f
		} else {
			node.Value[0] = 2.0 * (1 - f)
		}

	case "+":
		i0 := node.Args[0]
		i1 := node.Args[1]
		o0 := M.Nodes[i0.Node()].Value[i0.Output()]
		o1 := M.Nodes[i1.Node()].Value[i1.Output()]
		node.Value[0] = (o0 + o1) / 2.0

	case "-":
		i0, i1 := node.Args[0], node.Args[1]
		a := M.Nodes[i0.Node()].Value[i0.Output()]
		b := M.Nodes[i1.Node()].Value[i1.Output()]
		node.Value[0] = a - b

	case "*":
		i0, i1 := node.Args[0], node.Args[1]
		a := M.Nodes[i0.Node()].Value[i0.Output()]
		b := M.Nodes[i1.Node()].Value[i1.Output()]
		node.Value[0] = a * b

	case "/":
		i0, i1 := node.Args[0], node.Args[1]
		a := M.Nodes[i0.Node()].Value[i0.Output()]
		b := M.Nodes[i1.Node()].Value[i1.Output()]
		node.Value[0] = a / b

	case "max":
		i0 := node.Args[0]
		i1 := node.Args[1]
		p := M.Nodes[i0.Node()].Value[i0.Output()]
		q := M.Nodes[i1.Node()].Value[i1.Output()]
		if p > q {
			node.Value[0] = p
		} else {
			node.Value[0] = q
		}

	case "min":
		i0 := node.Args[0]
		i1 := node.Args[1]
		p := M.Nodes[i0.Node()].Value[i0.Output()]
		q := M.Nodes[i1.Node()].Value[i1.Output()]
		if p < q {
			node.Value[0] = p
		} else {
			node.Value[0] = q
		}

	case "and":
		i0 := node.Args[0]
		i1 := node.Args[1]
		p := M.Nodes[i0.Node()].Value[i0.Output()]
		q := M.Nodes[i1.Node()].Value[i1.Output()]
		if p > .5 && q > .5 {
			node.Value[0] = 1.0
		} else {
			node.Value[0] = 0.0
		}

	case "or":
		i0 := node.Args[0]
		i1 := node.Args[1]
		p := M.Nodes[i0.Node()].Value[i0.Output()]
		q := M.Nodes[i1.Node()].Value[i1.Output()]
		if p > .5 || q > .5 {
			node.Value[0] = 1.0
		} else {
			node.Value[0] = 0.0
		}

	case "xor":
		i0 := node.Args[0]
		i1 := node.Args[1]
		p := M.Nodes[i0.Node()].Value[i0.Output()]
		q := M.Nodes[i1.Node()].Value[i1.Output()]
		if p > .5 && q > .5 || p < .5 && q < .5 {
			node.Value[0] = 1.0
		} else {
			node.Value[0] = 0.0
		}

	case "noise":
		i0, i1 := node.Args[0], node.Args[1]
		p := M.Nodes[i0.Node()].Value[i0.Output()]
		q := M.Nodes[i1.Node()].Value[i1.Output()]
		node.Value[0] = .5 + pnoise.At2d(10*p, 10*q)

	case "lerp":
		i0, i1, i2 := node.Args[0], node.Args[1], node.Args[2]
		t := M.Nodes[i0.Node()].Value[i0.Output()]
		A := M.Nodes[i1.Node()].Value[i1.Output()]
		B := M.Nodes[i2.Node()].Value[i2.Output()]
		node.Value[0] = t*A + (1-t)*B

	case "if":
		i0, i1, i2 := node.Args[0], node.Args[1], node.Args[2]
		_cond := M.Nodes[i0.Node()].Value[i0.Output()]
		_then := M.Nodes[i1.Node()].Value[i1.Output()]
		_else := M.Nodes[i2.Node()].Value[i2.Output()]
		if _cond > .5 {
			node.Value[0] = _then
		} else {
			node.Value[0] = _else
		}

	default:
		msg := fmt.Sprintf("Op '%s' not implemented!", node.Op)
		panic(msg)
	}
	node.Ready = true
}

// Module //////////////////////////////////////////////////

func (M Module) Size() int {
	return len(M.Nodes)
}

func (M *Module) Clone() (newM *Module) {
	newM = &Module{
		Name:    M.Name,
		Nodes:   make([]*Node, len(M.Nodes)),
		Inputs:  make([]Port, len(M.Inputs)),
		Outputs: make([]Port, len(M.Outputs)),
	}
	for i := range M.Nodes {
		newM.Nodes[i] = M.Nodes[i].Clone()
	}
	copy(newM.Inputs, M.Inputs)
	copy(newM.Outputs, M.Outputs)
	return
}

func (M Module) String() string {
	s := "("
	for _, outp := range M.Outputs {
		s += fmt.Sprintf("%c", outp.Name)
	}
	s += ")"
	s += M.Name
	s += "("
	for _, inp := range M.Inputs {
		s += fmt.Sprintf("%c", inp.Name)
	}
	s += ")"
	s += "["
	for i, node := range M.Nodes {
		if i > 0 {
			s += "|"
		}
		colon := ""
		for j := range M.Outputs {
			if i == M.Outputs[j].Idx {
				s += fmt.Sprintf("%c", M.Outputs[j].Name)
				colon = ":"
			}
		}
		s += colon
		s += node.Op
		if node.Op == "=" {
			s += fmt.Sprintf(" %g", node.Value[0])
		} else {
			for _, arg := range node.Args {
				s += fmt.Sprintf(" %d", arg)
			}
		}
	}
	s += "]"
	return s
}

func (M Module) isInput(n int) bool {
	for i := range M.Inputs {
		if M.Inputs[i].Idx == n {
			return true
		}
	}
	return false
}

func (M Module) OutputNamesAsString() (s string) {
	for _, outp := range M.Outputs {
		s += fmt.Sprintf("%c", outp.Name)

	}
	return s
}

func (M Module) OutputIndices() (indices []int) {
	indices = make([]int, len(M.Outputs))
	for i := range M.Outputs {
		indices[i] = M.Outputs[i].Idx
	}
	return
}

func (M *Module) reconstructInputs() error {
	for i, inp := range M.Inputs {
		M.Inputs[i].Idx = -1
		name := fmt.Sprintf("%c", inp.Name)
		for j, node := range M.Nodes {
			if node.Op == name {
				if M.Inputs[i].Idx != -1 {
					return fmt.Errorf("Duplicate input '%s'", name)
				}
				M.Inputs[i].Idx = j
			}
		}
	}
	return nil
}

// Ordenación topológica: los nodos estan puestos de tal manera
// que no hay dependencias hacia nodos de menor índice.
// Esto permite evaluar con un bucle lineal desde los índices mayores
// a los menores.

type Topological []*_Node

func (t Topological) Len() int           { return len(t) }
func (t Topological) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t Topological) Less(i, j int) bool { return t[i].Order > t[j].Order }

func (M *Module) TopologicalSort() {
	_Nodes := make([]*_Node, len(M.Nodes))
	for i := range M.Nodes {
		_Nodes[i] = &_Node{
			Node:  *M.Nodes[i],
			Order: -1,
		}
	}
	changes := true
	for changes {
		changes = false
		for _, node := range _Nodes {
			if node.Order >= 0 {
				continue
			}
			max_child_order := 0 // for no-args nodes
			for _, arg := range node.Args {
				ord := _Nodes[arg.Node()].Order
				if ord == -1 {
					max_child_order = -1
					break
				}
				if ord > max_child_order {
					max_child_order = ord
				}
			}
			if max_child_order >= 0 {
				node.Order = max_child_order + 1
				changes = true
			}
		}
	}
	sorted_Nodes := make([]*_Node, len(_Nodes))
	for i := range _Nodes {
		sorted_Nodes[i] = _Nodes[i]
	}
	sort.Sort(Topological(sorted_Nodes))
	for i := range sorted_Nodes {
		sorted_Nodes[i].NewPos = i
	}
	for i := range sorted_Nodes {
		for j := range sorted_Nodes[i].Args {
			iold := sorted_Nodes[i].Args[j]
			inew := _Nodes[iold.Node()].NewPos
			sorted_Nodes[i].Args[j] = argument(inew, iold.Output())
		}
		M.Nodes[i] = &sorted_Nodes[i].Node
	}
	// Reconstruct Outputs
	for i := range M.Outputs {
		M.Outputs[i].Idx = _Nodes[M.Outputs[i].Idx].NewPos
	}
	M.reconstructInputs()
}

func (M *Module) TreeShake() {
	sz := M.Size()
	if sz == 0 {
		return
	}

	// First determine which nodes will be kept
	keep := make([]bool, sz)
	roots := M.OutputIndices()

	// make queue
	Q := make([]int, sz+1)
	for i := range Q {
		Q[i] = -1
	}
	top := 0

	qadd := func(i int) { // add to queue
		if find(i, Q) == -1 {
			Q[top] = i
			top++
		}
	}
	for i := range roots {
		qadd(roots[i])
	}
	curr := 0
	for Q[curr] != -1 {
		i := Q[curr]
		keep[i] = true
		for _, a := range M.Nodes[i].Args {
			qadd(a.Node())
		}
		curr++
	}

	// Assign new indices
	newindex := make([]int, sz)
	for i := range newindex {
		newindex[i] = -1
	}
	newi := 0
	for i := range M.Nodes {
		if !keep[i] {
			continue
		}
		newindex[i] = newi
		newi++
	}

	// Translate inputs + outputs
	for i := range M.Inputs {
		if M.Inputs[i].Idx != -1 {
			M.Inputs[i].Idx = newindex[M.Inputs[i].Idx]
		}
	}
	for i := range M.Outputs {
		if M.Outputs[i].Idx != -1 {
			M.Outputs[i].Idx = newindex[M.Outputs[i].Idx]
		}
	}

	// Keep nodes + translate indices
	keepnodes := []*Node{}
	for i, node := range M.Nodes {
		if !keep[i] {
			continue
		}
		for i := range node.Args {
			node.Args[i] = argument(newindex[node.Args[i].Node()], 0)
		}
		keepnodes = append(keepnodes, node)
	}
	M.Nodes = keepnodes
	return
}

func (M Module) SetInputs(inputs []float64) {
	for i := range M.Nodes {
		M.Nodes[i].Ready = false
	}
	for i := range M.Inputs {
		k := M.Inputs[i].Idx
		if k != -1 {
			M.Nodes[k].Value = []float64{inputs[i]}
			M.Nodes[k].Ready = true
		}
	}
}

func (M Module) GetOutputs() (outputs []float64) {
	for _, outp := range M.Outputs {
		outputs = append(outputs, M.Nodes[outp.Idx].Value[0])
	}
	return
}

func (M Module) inputIndex(name rune) (index int) {
	for i := range M.Inputs {
		if M.Inputs[i].Name == name {
			return i
		}
	}
	return -1
}

func (M Module) outputIndex(name rune) (index int) {
	for i := range M.Outputs {
		if M.Outputs[i].Name == name {
			return i
		}
	}
	return -1
}

func (M Module) EvalNodes(C *Circuit, roots ...int) {
	// Select nodes that we will compute
	selected := make([]int, M.Size())
	top := 0

	_add := func(i int) {
		if find(i, selected[:top]) == -1 {
			selected[top] = i
			top++
		}
	}

	for _, root := range roots {
		_add(root)
	}
	for i := 0; i < top; i++ {
		for _, arg := range M.Nodes[selected[i]].Args {
			_add(arg.Node())
		}
	}
	for i := top - 1; i >= 0; i-- {
		if M.Nodes[selected[i]].Call {
			node := &M.Nodes[selected[i]]
			inputs := []float64{}
			for _, arg := range (*node).Args {
				inputs = append(inputs, M.Nodes[arg.Node()].Value[arg.Output()])
			}
			outputs := C.EvalModule((*node).Op, inputs)
			(*node).Value[0] = outputs[0]
		} else {
			M.Nodes[selected[i]].eval(M)
		}
	}
}

func (mod Module) Eval(C *Circuit, inputs []float64) (outputs []float64) {
	mod.SetInputs(inputs)
	mod.EvalNodes(C, mod.OutputIndices()...)
	outputs = mod.GetOutputs()
	return
}

var rmodule = regexp.MustCompile(`\((.*)\)(.*)\((.*)\)\[(.*)\]`)

func parseModule(s string) (mod *Module, err error) {
	if len(s) == 0 {
		err = fmt.Errorf("Module is empty")
		return
	}
	match := rmodule.FindStringSubmatch(s)
	if len(match) != 5 {
		err = fmt.Errorf("Modules must have format `(abc)name(xyz)[...]`")
		return
	}
	outputs := match[1]
	name := match[2]
	inputs := match[3]
	body := match[4]

	if _, ok := OperatorInfo[name]; ok {
		return mod, fmt.Errorf("Module name '%s' is reserved", name)
	}
	mod = &Module{Name: name}

	for _, c := range inputs {
		mod.Inputs = append(mod.Inputs, Port{Name: c, Idx: -1})
	}
	for _, c := range outputs {
		mod.Outputs = append(mod.Outputs, Port{Name: c, Idx: -1})
	}

	for i, snod := range strings.Split(body, "|") {
		parts := strings.Split(snod, ":")
		switch len(parts) {
		case 1:
			snod = parts[0]
		case 2:
			for _, c := range strings.TrimSpace(parts[0]) {
				k := mod.outputIndex(c)
				if k == -1 {
					err = fmt.Errorf("There is no output '%c'", c)
					return
				}
				mod.Outputs[k].Idx = i
			}
			snod = parts[1]
		default:
			err = fmt.Errorf("Error in node %d: wrong number of ':'", i)
			return
		}

		if snod == "" {
			err = fmt.Errorf("Empty node")
			return
		}

		var op string
		rnod := strings.NewReader(snod)
		n, err2 := fmt.Fscanf(rnod, "%s", &op)
		if n != 1 || err2 != nil {
			err = fmt.Errorf("Error in node %d: '%s'", i, snod)
			return
		}

		var node *Node

		if info, ok := OperatorInfo[op]; !ok {
			// An input or a call
			node = &Node{
				Op:    op,
				Value: []float64{0.0},
			}
			k := mod.inputIndex(rune(op[0]))
			if k != -1 { // An input
				if mod.Inputs[k].Idx != -1 {
					err = fmt.Errorf("Duplicated input '%c'", mod.Inputs[k].Name)
					return
				}
				mod.Inputs[k].Idx = i
			} else {
				for {
					var arg int
					n, err := fmt.Fscanf(rnod, "%d", &arg)
					if n != 1 || err != nil {
						break
					}
					node.Args = append(node.Args, argument(arg/10, arg%10))
				}
			}
		} else {
			if op == "=" {
				// A constant
				var val float64
				n, _ := fmt.Fscanf(rnod, "%f", &val)
				if n != 1 {
					err = fmt.Errorf("Error in node %d: cannot read constant", i)
					return
				}
				node = &Node{
					Op:    "=",
					Value: []float64{val},
				}
			} else {
				// An operator
				node = &Node{
					Op:    op,
					Value: []float64{0.0},
				}
				for {
					var arg int
					n, err := fmt.Fscanf(rnod, "%d", &arg)
					if n != 1 || err != nil {
						break
					}
					node.Args = append(node.Args, argument(arg/10, arg%10))
				}
				if info.Nargs != len(node.Args) {
					err = fmt.Errorf("Error in node %d: `%s` has %d args, not %d.",
						i, op, info.Nargs, len(node.Args))
					return
				}
			}
		}
		mod.Nodes = append(mod.Nodes, node)
	}
	if len(mod.Outputs) == 0 {
		err = fmt.Errorf("Error in module: there are no outputs")
		return
	}

	// check missing outputs
	for i := range mod.Outputs {
		if mod.Outputs[i].Idx == -1 {
			err = fmt.Errorf("Missing output `%c`", mod.Outputs[i].Name)
			return
		}
	}

	// check missing nodes
	for i, node := range mod.Nodes {
		for j, arg := range node.Args {
			if arg.IsUnset() {
				err = fmt.Errorf("Argument %d missing in node '%d'", j, i)
				return
			}
			if arg.Node() >= len(mod.Nodes) {
				err = fmt.Errorf("Nonexistent node %d", arg.Node())
				return
			}
		}
	}
	return
}

func readModule(s string) (mod *Module, err error) {
	mod, err = parseModule(s)
	if err != nil {
		return
	}
	mod.TopologicalSort()
	mod.TreeShake()
	return
}

func RandomModule(inputs, outputs string, numnodes int) (M Module) {
	for _, c := range inputs {
		M.Inputs = append(M.Inputs, Port{Name: c, Idx: -1})
	}
	for _, c := range outputs {
		M.Outputs = append(M.Outputs, Port{Name: c, Idx: -1})
	}
	// Add Input nodes
	for i := range M.Inputs {
		M.Nodes = append(M.Nodes, &Node{
			Op:    fmt.Sprintf("%c", M.Inputs[i].Name),
			Value: []float64{0.0},
		})
	}
	// Generate nodes
	curr := len(M.Inputs)
	for i := 0; i < numnodes; i++ {
		iop := rand.Intn(len(Operators))
		op := Operators[iop]
		info := OperatorInfo[op]
		args := []Argument{}
		val := 0.0
		if op == "=" {
			val = rand.Float64()
		} else {
			for _, a := range rand.Perm(curr)[:info.Nargs] {
				args = append(args, argument(a, 0))
			}
		}
		M.Nodes = append(M.Nodes, &Node{
			Op:    op,
			Args:  args,
			Value: []float64{val},
		})
		curr++
	}
	// Assign outputs
	for i := range M.Outputs {
		M.Outputs[i].Idx = rand.Intn(curr)
	}
	M.reconstructInputs()
	M.TopologicalSort()
	M.TreeShake()
	return
}

func RandomModule2(inputs, outputs string, numnodes int) (M *Module) {
	M = &Module{}
	for _, c := range inputs {
		M.Inputs = append(M.Inputs, Port{Name: c, Idx: -1})
	}
	for _, c := range outputs {
		M.Outputs = append(M.Outputs, Port{Name: c, Idx: -1})
	}

	// 1) Generate nodes without connections
	for i := 0; i < numnodes; i++ {
		iop := rand.Intn(len(Operators))
		op := Operators[iop]
		info := OperatorInfo[op]
		args := []Argument{}
		val := 0.0
		if op == "=" {
			val = rand.Float64()
		} else {
			for i := 0; i < info.Nargs; i++ {
				args = append(args, -1)
			}
		}
		M.Nodes = append(M.Nodes, &Node{
			Op:    op,
			Args:  args,
			Value: []float64{val},
		})
	}
	for i := range M.Inputs { // + add Input nodes at the end
		k := len(M.Nodes)
		M.Nodes = append(M.Nodes, &Node{
			Op:    fmt.Sprintf("%c", M.Inputs[i].Name),
			Value: []float64{0.0},
		})
		M.Inputs[i].Idx = k
	}

	// 2) Set the output of every node to a node below
	for i := range M.Nodes {
		// how many inputs below
		ninputs := 0
		for j := 0; j < i; j++ {
			for k := range M.Nodes[j].Args {
				if M.Nodes[j].Args[k] == -1 {
					ninputs++
				}
			}
		}
		noutputs := 0
		for j := range M.Outputs {
			if M.Outputs[j].Idx == -1 {
				noutputs++
			}
		}

		if ninputs == 0 && noutputs == 0 {
			continue
		}

		r := rand.Intn(ninputs + noutputs)

		if r >= ninputs {
			// assign to output
			r -= ninputs
			for j := range M.Outputs {
				if M.Outputs[j].Idx == -1 {
					if r--; r < 0 {
						M.Outputs[j].Idx = i
						goto done
					}
				}
			}
			panic("unreachable1")
		} else {
			// assign to input of other node
			for j := 0; j < i; j++ {
				for k := range M.Nodes[j].Args {
					if M.Nodes[j].Args[k] == -1 {
						if r--; r < 0 {
							M.Nodes[j].Args[k] = argument(i, 0)
							goto done
						}
					}
				}
			}
			panic("didn't assign output!")
		}
	done:
	}

	// 3) Assign at random the remaining links
	sz := len(M.Nodes)
	for i := range M.Nodes {
		for j, a := range M.Nodes[i].Args {
			if a == -1 {
				M.Nodes[i].Args[j] = argument(i+1+rand.Intn(sz-i-1), 0)
			}
		}
	}

	// 4) Assign at random the remaining outputs
	for i := range M.Outputs {
		if M.Outputs[i].Idx == -1 {
			M.Outputs[i].Idx = rand.Intn(sz)
		}
	}
	M.reconstructInputs()
	M.TopologicalSort()
	M.TreeShake()
	return
}

var (
	OperatorChangeProbability = 1.0
	ConnectionSwapProbability = 0.5
)

func (M *Module) Mutate() {
	r := rand.Float64()
	r -= OperatorChangeProbability
	if r < 0 {
		M.MutOperatorChange()
	}
	r -= ConnectionSwapProbability
	if r < 0 {
		M.MutConnectionSwap()
	}
}

type Queue struct {
	elems     []int
	curr, top int
}

func NewQueue(maxsize int) (Q Queue) {
	Q.elems = make([]int, maxsize+1)
	for i := range Q.elems {
		Q.elems[i] = -1
	}
	Q.top = 0
	Q.curr = 0
	return
}

func (Q *Queue) Empty() bool { return Q.curr == Q.top }
func (Q *Queue) Next()       { Q.curr++ }
func (Q *Queue) Curr() int   { return Q.elems[Q.curr] }

func (Q *Queue) Find(x int) int {
	for i := range Q.elems {
		if Q.elems[i] == x {
			return i
		}
	}
	return -1
}

func (Q *Queue) Add(x int) {
	if Q.Find(x) == -1 {
		Q.elems[Q.top] = x
		Q.top++
	}
}

func (M Module) MarkInputsOf(n int) (marks []bool) {
	marks = make([]bool, len(M.Nodes))
	Q := NewQueue(len(M.Nodes))
	Q.Add(n)
	for !Q.Empty() {
		i := Q.Curr()
		marks[i] = true
		for _, a := range M.Nodes[i].Args {
			Q.Add(a.Node())
		}
		Q.Next()
	}
	return
}

type Link struct {
	Node, Input int
}

func swap(a, b *Argument) {
	*a, *b = *b, *a
}

func (M *Module) MutConnectionSwap() {

	for tries := 3; tries > 0; tries-- {
		// escoger al azar 2 links
		links1 := []Link{}
		for i := range M.Nodes {
			for j := range M.Nodes[i].Args {
				links1 = append(links1, Link{Node: i, Input: j})
			}
		}
		sz1 := len(links1)
		if sz1 == 0 {
			continue
		}
		L1 := links1[rand.Intn(sz1)]

		marks := M.MarkInputsOf(L1.Node)

		links2 := []Link{}
		for i := range M.Nodes {
			if marks[i] { // avoid predecessors of L1.Node to avoid creating loops
				continue
			}
			for j := range M.Nodes[i].Args {
				links2 = append(links2, Link{Node: i, Input: j})
			}
		}

		sz2 := len(links2)
		if sz2 == 0 {
			continue
		}
		L2 := links2[rand.Intn(sz2)]

		// swap
		swap(
			&M.Nodes[L1.Node].Args[L1.Input],
			&M.Nodes[L2.Node].Args[L2.Input],
		)
		M.TopologicalSort()
		return
	}
}

func (M *Module) MutOperatorChange() {
	candidates := []int{}
	for i := range M.Nodes {
		op := M.Nodes[i].Op
		info := OperatorInfo[op]
		if info.Nargs >= 1 && info.Nargs <= 2 {
			candidates = append(candidates, i)
		}
	}
	k := candidates[rand.Intn(len(candidates))]
	chosen := M.Nodes[k].Op
	info := OperatorInfo[chosen]
	nargs := info.Nargs
	alternatives := []string{}
	same_args := NumArguments[nargs]
	for i := range same_args {
		if same_args[i] != chosen {
			alternatives = append(alternatives, same_args[i])
		}
	}
	M.Nodes[k].Op = alternatives[rand.Intn(len(alternatives))]
}

// Circuit /////////////////////////////////////////////////

func (C Circuit) EvalModule(name string, inputs []float64) (outputs []float64) {
	mod, ok := C.Modules[name]
	if !ok {
		msg := fmt.Sprintf("Module '%s' missing", mod)
		panic(msg)
	}
	mod.SetInputs(inputs)
	mod.EvalNodes(&C, mod.OutputIndices()...)
	outputs = mod.GetOutputs()
	return
}

func (C Circuit) Eval(inputs []float64) (outputs []float64) {
	return C.EvalModule("", inputs)
}

func _map(x float64) (y float64) {
	y = x
	if y > 1.0 {
		y = 1.0
	}
	if y < 0.0 {
		y = 0.0
	}
	return
}

func (C Circuit) RenderPixel(xlow, ylow, xhigh, yhigh float64, samples int) Color {
	xsz := (xhigh - xlow) / float64(samples)
	ysz := (yhigh - ylow) / float64(samples)
	S := make([]float64, samples*2)
	for i := 0; i < samples; i++ {
		S[i*2] = xlow + float64(i)*xsz + xsz*rand.Float64()
		S[i*2+1] = ylow + float64(i)*ysz + ysz*rand.Float64()
	}
	for dim := 0; dim < 2; dim++ {
		for i := 0; i < samples; i++ {
			_i := rand.Intn(samples)
			S[i*2+dim], S[_i*2+dim] = S[_i*2+dim], S[i*2+dim]
		}
	}
	var c Color
	for i := 0; i < len(S); i += 2 {
		x, y := S[i], S[i+1]
		_x, _y := x-.5, y-.5
		r := math.Sqrt(_x*_x + _y*_y)
		t := math.Atan2(_y, _x)/(2.0*math.Pi) + .5
		inputs := []float64{x, y, r, t}
		out := C.Eval(inputs)
		c.Add(Color{out[0], out[1], out[2]})
	}
	return c.Divide(float64(samples))
}

func (C Circuit) Render(size, samples int) image.Image {
	img := NewImage(size, size)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			xlow := float64(i) / float64(size)
			xhigh := float64(i+1) / float64(size)
			ylow := float64(j) / float64(size)
			yhigh := float64(j+1) / float64(size)
			px := C.RenderPixel(xlow, ylow, xhigh, yhigh, samples)
			img.px[i][j] = color.RGBA{
				uint8(_map(px.R) * 255.0),
				uint8(_map(px.G) * 255.0),
				uint8(_map(px.B) * 255.0),
				255,
			}
		}
	}
	return img
}

func (C Circuit) Clone() (newC Circuit) {
	newC.Modules = make(map[string]*Module)
	for name, mod := range C.Modules {
		newC.Modules[name] = mod.Clone()
	}
	return
}

func RandomCircuit(numnodes int) (C Circuit) {
	C.Modules = make(map[string]*Module)
	C.Modules[""] = RandomModule2("xyrt", "rgb", numnodes)
	return C
}

func (C Circuit) Mutate() {
	C.Modules[""].MutConnectionSwap()
}

func (C Circuit) String() (s string) {
	i := 0
	for _, mod := range C.Modules {
		if i > 0 {
			s += ";"
		}
		s += mod.String()
		i++
	}
	return
}

func Read(s string) (C Circuit, err error) {
	C.Modules = make(map[string]*Module)
	smodules := strings.Split(s, ";")
	for _, smod := range smodules {
		mod, err := parseModule(smod)
		if err != nil {
			return C, err
		}
		mod.TopologicalSort()
		mod.TreeShake()
		if _, ok := C.Modules[mod.Name]; !ok {
			C.Modules[mod.Name] = mod
		} else {
			return C, fmt.Errorf("Duplicated module `%s`.", mod.Name)
		}
	}
	// Checks:
	// 1) There is a main module, with an empty name.
	main, ok := C.Modules[""]
	if !ok {
		return C, fmt.Errorf("There is no main module (with empty name)")
	}
	// 2) The main module has rgb as outputs.
	if names := main.OutputNamesAsString(); names != "rgb" {
		return C, fmt.Errorf("Outputs != 'rgb'! (outputs = '%s')", names)
	}
	// 3) All modules except main have 1 output
	for name, module := range C.Modules {
		if module.Name != "" && len(module.Outputs) != 1 {
			return C, fmt.Errorf("Module `%s` has more than one output", name)
		}
	}

	// Determine which nodes are calls to other modules
	// + check number of args is correct
	for name, mod := range C.Modules {
		isInput := make(map[string]bool)
		for i := range mod.Inputs {
			sname := fmt.Sprintf("%c", mod.Inputs[i].Name)
			isInput[sname] = true
		}
		for i, node := range mod.Nodes {
			_, isOperator := OperatorInfo[node.Op]
			if isOperator || isInput[node.Op] {
				continue
			}
			if _, ok := C.Modules[node.Op]; ok {
				C.Modules[name].Nodes[i].Call = true
				has := len(C.Modules[node.Op].Inputs)
				used := len(node.Args)
				if used != has {
					err = fmt.Errorf("Module `%s` has %d inputs, not %d.", node.Op, has, used)
					return
				}
			} else {
				err = fmt.Errorf("Missing module `%s`", node.Op)
				return
			}
		}
	}
	return
}

func (C Circuit) Graphviz(w io.Writer) {
	fmt.Fprintf(w, "digraph Circuit {\n")
	for name, mod := range C.Modules {
		if name == "" {
			name = "main"
		}
		fmt.Fprintf(w, "   subgraph %s {\n", name)

		// Inputs
		fmt.Fprintf(w, "      { rank = same;\n")
		for _, port := range mod.Inputs {
			fmt.Fprintf(w, `      %d [label="%c",shape=square,style=filled];`,
				port.Idx, port.Name)
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "      }\n")

		// Outputs
		fmt.Fprintf(w, "      { rank = same;\n")
		for i, _ := range mod.Outputs {
			k := len(mod.Nodes) + i
			fmt.Fprintf(w, "      %d [label=\"%c\",shape=square,style=filled];\n",
				k, mod.Outputs[i].Name)
		}
		fmt.Fprintf(w, "      }\n")

		// Middle nodes
		for i, node := range mod.Nodes {
			if mod.isInput(i) {
				continue
			}
			if node.Op == "=" {
				fmt.Fprintf(w, `      %d [label="%.2f",shape=diamond,style=filled,color="#99aaff"]`,
					i, node.Value)
			} else {
				fmt.Fprintf(w, `      %d [label="%s"];`, i, node.Op)
			}
			fmt.Fprintln(w)
		}

		// Links
		for i, node := range mod.Nodes {
			for _, arg := range node.Args {
				fmt.Fprintf(w, `      %d -> %d;`, arg.Node(), i)
				fmt.Fprintln(w)
			}
		}
		for i, out := range mod.Outputs {
			k := len(mod.Nodes) + i
			fmt.Fprintf(w, "      %d -> %d;\n", out.Idx, k)
		}
		fmt.Fprintf(w, "   }\n")
	}
	fmt.Fprintf(w, "}\n")
}

// Image ///////////////////////////////////////////////////

type Image struct {
	h, w int
	px   [][]color.RGBA
}

func (I *Image) At(x, y int) color.Color { return I.px[x][y] }
func (I *Image) ColorModel() color.Model { return color.RGBAModel }
func (I *Image) Bounds() image.Rectangle { return image.Rect(0, 0, I.h, I.w) }

func NewImage(h, w int) *Image {
	px := make([][]color.RGBA, h)
	for i := range px {
		px[i] = make([]color.RGBA, w)
	}
	return &Image{h, w, px}
}
