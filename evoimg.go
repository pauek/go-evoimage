package evoimage

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	// "sync"
)

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
	1: {"=", "id", "cos", "sin", "inv", "band", "bw", "not"},
	2: {"+", "*", "and", "or", "xor"},
	3: {"lerp", "if"},
}

var OperatorInfo = make(map[string]OpInfo)

func init() {
	for nargs, ops := range NumArguments {
		for _, op := range ops {
			OperatorInfo[op] = OpInfo{
				Nargs: nargs,
			}
		}
	}
}

type Color struct {
	R, G, B float64
}
type Node struct {
	Op    string
	Args  []int
	Value float64
	Ready bool
}
type _Node struct {
	Node
	Order  int
	NewPos int
}
type Module struct {
	Name        string
	Nodes       []*Node
	Inputs      [][]int
	InputNames  []rune
	Outputs     []int
	OutputNames []rune
}

func (c *Color) Add(other Color) {
	c.R += other.R
	c.G += other.G
	c.B += other.B
}

func (c *Color) Divide(x float64) Color {
	return Color{c.R / x, c.G / x, c.B / x}
}

// Ordenación topológica: los nodos estan puestos de tal manera
// que no hay dependencias hacia nodos de menor índice.
// Esto permite evaluar con un bucle lineal desde los índices mayores
// a los menores.

type Topological []*_Node

func (t Topological) Len() int           { return len(t) }
func (t Topological) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t Topological) Less(i, j int) bool { return t[i].Order > t[j].Order }

func (M Module) Size() int {
	return len(M.Nodes)
}

func (M Module) TopologicalSort() {
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
				ord := _Nodes[arg].Order
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
			inew := _Nodes[iold].NewPos
			sorted_Nodes[i].Args[j] = inew
		}
		M.Nodes[i] = &sorted_Nodes[i].Node
	}
}

func (M Module) TreeShake(roots ...int) (newM Module) {
	sz := M.Size()
	if sz == 0 {
		return
	}
	inputs := make([][]int, len(M.Inputs))
	inputNames := make([]rune, len(M.InputNames))
	newM.Outputs = make([]int, len(M.Outputs))
	newM.OutputNames = make([]rune, len(M.OutputNames))

	order := make([]int, sz+1)
	for i := range order {
		order[i] = -1
	}
	top := 0
	for _, root := range uniq(roots) {
		order[top] = root
		top++
	}
	curr := 0
	for order[curr] != -1 {
		i := order[curr]
		node := M.Nodes[i]
		newnode := Node{
			Op:    node.Op,
			Args:  make([]int, len(node.Args)),
			Value: node.Value,
		}
		for j := i + 1; j < sz; j++ {
			if k := find(j, node.Args); k != -1 {
				arg := j
				l := find(arg, order)
				if l == -1 {
					newnode.Args[k] = top
					order[top] = arg
					top++
				} else {
					newnode.Args[k] = l
				}
			}
		}
		newM.Nodes = append(newM.Nodes, &newnode)
		for j := range M.Outputs {
			if i == M.Outputs[j] {
				newM.Outputs[j] = curr
				newM.OutputNames[j] = M.OutputNames[j]
			}
		}
		for j := range M.Inputs {
			for k := range M.Inputs[j] {
				if i == M.Inputs[j][k] {
					inputs[j] = append(inputs[j], curr)
					inputNames[j] = M.InputNames[j]
				}
			}
		}
		curr++
	}
	fmt.Println(inputs, inputNames)
	// remove unused inputs (detect with empty inputnames)
	for i := range inputs {
		if len(inputs[i]) > 0 {
			newM.Inputs = append(newM.Inputs, inputs[i])
			newM.InputNames = append(newM.InputNames, inputNames[i])
		}
	}
	return
}

func (node *Node) eval(M Module) {
	if node.Ready {
		return
	}
	switch node.Op {
	case "=":
		// Value is already there
	case "id":
		node.Value = M.Nodes[node.Args[0]].Value
	case "+":
		a := M.Nodes[node.Args[0]].Value
		b := M.Nodes[node.Args[1]].Value
		node.Value = (a + b) / 2.0
	case "*":
		a := M.Nodes[node.Args[0]].Value
		b := M.Nodes[node.Args[1]].Value
		node.Value = a * b
	case "cos":
		f := M.Nodes[node.Args[0]].Value
		node.Value = (1 + math.Cos(2*math.Pi*f)) / 2
	case "sin":
		f := M.Nodes[node.Args[0]].Value
		node.Value = (1 + math.Sin(2*math.Pi*f)) / 2
	case "lerp":
		t := M.Nodes[node.Args[0]].Value
		A := M.Nodes[node.Args[1]].Value
		B := M.Nodes[node.Args[2]].Value
		node.Value = t*A + (1-t)*B
	case "inv":
		a := M.Nodes[node.Args[0]].Value
		node.Value = (1 - a)
	case "band":
		a := M.Nodes[node.Args[0]].Value
		if a > .33 && a < .66 {
			node.Value = 1.0
		} else {
			node.Value = 0.0
		}
	case "bw":
		a := M.Nodes[node.Args[0]].Value
		if a > .5 {
			node.Value = 1.0
		} else {
			node.Value = 0.0
		}
	case "and":
		p := M.Nodes[node.Args[0]].Value
		q := M.Nodes[node.Args[1]].Value
		if p > .5 && q > .5 {
			node.Value = 1.0
		} else {
			node.Value = 0.0
		}
	case "or":
		p := M.Nodes[node.Args[0]].Value
		q := M.Nodes[node.Args[1]].Value
		if p > .5 || q > .5 {
			node.Value = 1.0
		} else {
			node.Value = 0.0
		}
	case "xor":
		p := M.Nodes[node.Args[0]].Value
		q := M.Nodes[node.Args[1]].Value
		if p > .5 && q > .5 ||
			p < .5 && q < .5 {
			node.Value = 1.0
		} else {
			node.Value = 0.0
		}
	case "not":
		p := M.Nodes[node.Args[0]].Value
		if p > .5 {
			node.Value = 0.0
		} else {
			node.Value = 1.0
		}
	case "if":
		cond := M.Nodes[node.Args[0]].Value
		_then := M.Nodes[node.Args[1]].Value
		_else := M.Nodes[node.Args[2]].Value
		if cond > .5 {
			node.Value = _then
		} else {
			node.Value = _else
		}
	default:
		msg := fmt.Sprintf("Op '%s' not implemented!", node.Op)
		panic(msg)
	}
	node.Ready = true
}

func (M Module) SetInputs(inputs []float64) {
	for i := range M.Nodes {
		M.Nodes[i].Ready = false
	}
	for i := range inputs {
		for j := range M.Inputs[i] {
			node := M.Nodes[M.Inputs[i][j]]
			node.Value = inputs[i]
			node.Ready = true
		}
	}
}

func (M Module) GetOutputs() (outputs []float64) {
	for i := range M.Outputs {
		outputs = append(outputs, M.Nodes[M.Outputs[i]].Value)
	}
	return
}

func (M Module) inputIndex(name rune) (index int) {
	for i := range M.Inputs {
		if M.InputNames[i] == name {
			return i
		}
	}
	return -1
}

func (M Module) outputIndex(name rune) (index int) {
	for i := range M.Outputs {
		if M.OutputNames[i] == name {
			return i
		}
	}
	return -1
}

func (M Module) EvalNodes(roots ...int) {
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
			_add(arg)
		}
	}
	for i := top - 1; i >= 0; i-- {
		M.Nodes[selected[i]].eval(M)
	}
}

func (M Module) Eval(inputs []float64) (outputs []float64) {
	M.SetInputs(inputs)
	M.EvalNodes(M.Outputs...)
	outputs = M.GetOutputs()
	return
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

func (M Module) RenderPixel(xlow, ylow, xhigh, yhigh float64, samples int) Color {
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
		out := M.Eval([]float64{S[i], S[i+1]})
		c.Add(Color{out[0], out[1], out[2]})
	}
	return c.Divide(float64(samples))
}

func (M Module) Render(size, samples int) image.Image {
	// var wg sync.WaitGroup
	img := NewImage(size, size)
	// wg.Add(size)
	for i := 0; i < size; i++ {
		// func(i int) {
		for j := 0; j < size; j++ {
			xlow := float64(i) / float64(size)
			xhigh := float64(i+1) / float64(size)
			ylow := float64(j) / float64(size)
			yhigh := float64(j+1) / float64(size)
			c := M.RenderPixel(xlow, ylow, xhigh, yhigh, samples)
			img.px[i][j] = color.RGBA{
				uint8(_map(c.R) * 255.0),
				uint8(_map(c.G) * 255.0),
				uint8(_map(c.B) * 255.0),
				255,
			}
		}
		// wg.Done()
		// }(i)
	}
	// wg.Wait()
	return img
}

func (M Module) String() string {
	s := "("
	for _, c := range M.OutputNames {
		s += fmt.Sprintf("%c", c)
	}
	s += ")"
	s += M.Name
	s += "("
	for _, c := range M.InputNames {
		s += fmt.Sprintf("%c", c)
	}
	s += ")"
	s += "["
	for i, node := range M.Nodes {
		if i > 0 {
			s += "|"
		}
		colon := ""
		for j := range M.Outputs {
			if i == M.Outputs[j] {
				s += fmt.Sprintf("%c", M.OutputNames[j])
				colon = ":"
			}
		}
		s += colon
		s += node.Op
		if node.Op == "=" {
			s += fmt.Sprintf(" %g", node.Value)
		} else {
			for _, arg := range node.Args {
				s += fmt.Sprintf(" %d", arg)
			}
		}
	}
	s += "]"
	return s
}

var rmodule = regexp.MustCompile(`\((.*)\)([_a-zA-Z]*)\((.*)\)\[(.*)\]`)

func readModule(s string) (mod Module, err error) {
	if len(s) == 0 {
		err = fmt.Errorf("Module is empty")
		return
	}
	match := rmodule.FindStringSubmatch(s)
	if len(match) != 5 {
		err = fmt.Errorf("Modules must have a format like `(abc)name(xyz)[...]`")
		return
	}
	outputs := match[1]
	name := match[2]
	inputs := match[3]
	body := match[4]

	mod.Name = name
	for _, c := range inputs {
		mod.Inputs = append(mod.Inputs, []int{})
		mod.InputNames = append(mod.InputNames, c)
	}
	for _, c := range outputs {
		mod.Outputs = append(mod.Outputs, -1)
		mod.OutputNames = append(mod.OutputNames, c)
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
				mod.Outputs[k] = i
			}
			snod = parts[1]
		default:
			err = fmt.Errorf("Error in node %d: wrong number of ':'", i)
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
			// An input
			k := mod.inputIndex(rune(op[0]))
			if k == -1 {
				err = fmt.Errorf("node '%s' is not an operator nor an input", op)
				return
			}
			node = &Node{Op: op}
			mod.Inputs[k] = append(mod.Inputs[k], i)
		} else {
			if op == "=" {
				// A constant
				node = &Node{Op: "="}
				n, _ := fmt.Fscanf(rnod, "%f", &node.Value)
				if n != 1 {
					err = fmt.Errorf("Error in node %d: cannot read constant", i)
					return
				}
			} else {
				// An operator
				node = &Node{Op: op}
				for {
					var arg int
					n, err := fmt.Fscanf(rnod, "%d", &arg)
					if n != 1 || err != nil {
						break
					}
					node.Args = append(node.Args, arg)
				}
				if info.Nargs != len(node.Args) {
					err = fmt.Errorf("Error in node %d: '%s' should have %d args", i, op, info.Nargs)
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
	return
}

func Read(s string) (mod Module, err error) {
	mod, err = readModule(s)
	if err != nil {
		return
	}
	mod.TopologicalSort()
	mod = mod.TreeShake(mod.Outputs...)
	return
}

// Image

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
