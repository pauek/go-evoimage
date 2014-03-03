package evoimage

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
)

const (
	// sources
	Const = iota
	X
	Y
	R
	T

	// unary
	Id
	Sin
	Cos
	Inv
	Band
	Bw
	Not

	// binary
	Sum
	Mult
	And
	Or
	Xor
	Blur

	// ternary
	Lerp
	If
	Map
)

const MAX_ARGS = 10

type OpInfo struct {
	Name  string
	Nargs int
}

var OperatorInfo = map[int]OpInfo{
	Const: {"=", 1},
	X:     {"x", 0},
	Y:     {"y", 0},
	R:     {"r", 0},
	T:     {"t", 0},

	Id:   {"id", 1},
	Cos:  {"cos", 1},
	Sin:  {"sin", 1},
	Inv:  {"inv", 1},
	Band: {"band", 1},
	Bw:   {"bw", 1},
	Not:  {"not", 1},

	Sum:  {"+", 2},
	Mult: {"*", 2},
	And:  {"and", 2},
	Or:   {"or", 2},
	Xor:  {"xor", 2},
	Blur: {"blur", 2}, // (blur <img> <blur-radius>)

	Lerp: {"lerp", 3},
	If:   {"if", 3},
	Map:  {"map", 3},
}

var Ids = map[string]int{}

func init() {
	for id, info := range OperatorInfo {
		Ids[info.Name] = id
	}
}

type Color struct {
	R, G, B float64
}
type Node struct {
	Op    int
	Args  []int
	Const float64
}
type _Node struct {
	Node
	Order  int
	NewPos int
}
type Expression struct {
	Nodes   []*Node
	R, G, B int
}

func (N *Node) Name() string {
	return OperatorInfo[N.Op].Name
}

func (E Expression) Size() int {
	return len(E.Nodes)
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

type Topological []*_Node

func (t Topological) Len() int           { return len(t) }
func (t Topological) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t Topological) Less(i, j int) bool { return t[i].Order > t[j].Order }

func (E Expression) TopologicalSort() {
	_Nodes := make([]*_Node, len(E.Nodes))
	for i := range E.Nodes {
		_Nodes[i] = &_Node{
			Node:  *E.Nodes[i],
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
		E.Nodes[i] = &sorted_Nodes[i].Node
	}
}

func (E Expression) TreeShake(roots ...int) Expression {
	sz := E.Size()
	if sz == 0 {
		return Expression{}
	}
	order := make([]int, sz+1)
	for i := range order {
		order[i] = -1
	}
	top := 0
	for _, root := range uniq(roots) {
		order[top] = root
		top++
	}
	var newE Expression
	curr := 0
	for order[curr] != -1 {
		i := order[curr]
		node := E.Nodes[i]
		newnode := Node{
			Op:    node.Op,
			Args:  make([]int, len(node.Args)),
			Const: node.Const,
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
		newE.Nodes = append(newE.Nodes, &newnode)
		switch {
		case i == E.R:
			newE.R = curr
		case i == E.G:
			newE.G = curr
		case i == E.B:
			newE.B = curr
		}
		curr++
	}
	return newE
}

func (node *Node) eval(E Expression, x, y float64, args []float64) float64 {
	switch node.Op {
	case Const:
		return node.Const
	case Id:
		return args[0]
	case X:
		return x
	case Y:
		return y
	case R:
		_x, _y := 2*(x-.5), 2*(y-.5)
		return math.Sqrt(_x*_x + _y*_y)
	case T:
		_x, _y := x-.5, math.Abs(y-.5)
		return math.Atan2(_y, -_x) / math.Pi
	case Sum:
		return (args[0] + args[1]) / 2.0
	case Mult:
		return args[0] * args[1]
	case Cos:
		return (1 + math.Cos(2*math.Pi*args[0])) / 2
	case Sin:
		return (1 + math.Sin(2*math.Pi*args[0])) / 2
	case Lerp:
		return args[0]*args[1] + (1-args[0])*args[2]
	case Inv:
		return (1 - args[0])
	case Band:
		if args[0] > .33 && args[0] < .66 {
			return 1.0
		} else {
			return 0.0
		}
	case Bw:
		if args[0] > .5 {
			return 1.0
		} else {
			return 0.0
		}
	case And:
		if args[0] > .5 && args[1] > .5 {
			return 1.0
		} else {
			return 0.0
		}
	case Or:
		if args[0] > .5 || args[1] > .5 {
			return 1.0
		} else {
			return 0.0
		}
	case Xor:
		if args[0] > .5 && args[1] > .5 ||
			args[0] < .5 && args[1] < .5 {
			return 1.0
		} else {
			return 0.0
		}
	case Not:
		if args[0] > .5 {
			return 0.0
		} else {
			return 1.0
		}
	case If:
		if args[0] > .5 {
			return args[1]
		} else {
			return args[2]
		}
	case Blur:
		v := 0.0
		radius := MAX_BLUR_RADIUS * args[1]
		for i := 0; i < BLUR_SAMPLES; i++ {
			dx := radius * (2.0*rand.Float64() - 1.0)
			dy := radius * (2.0*rand.Float64() - 1.0)
			v += E.EvalNodes(x+dx, y+dy, node.Args[0])[node.Args[0]]
		}
		return v / float64(BLUR_SAMPLES)

	case Map:
		_x, _y := args[1], args[2]
		return E.EvalNodes(_x, _y, node.Args[0])[node.Args[0]]

	default:
		panic("not implemented")
	}
}

const BLUR_SAMPLES = 5
const MAX_BLUR_RADIUS = 0.05

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

func (E Expression) EvalNodes(x, y float64, roots ...int) []float64 {
	// Select nodes that we will compute
	selected := make([]int, E.Size())
	top := 0
	for _, root := range uniq(roots) {
		selected[top] = root
		top++
	}
	for i := 0; i < top; i++ {
		node := E.Nodes[selected[i]]
		for _, arg := range node.Args {
			if find(arg, selected) == -1 {
				selected[top] = arg
				top++
			}
		}
	}
	values := make([]float64, E.Size())
	args := make([]float64, MAX_ARGS)
	for i := top - 1; i >= 0; i-- {
		node := E.Nodes[selected[i]]
		for j, arg := range node.Args {
			args[j] = values[arg]
		}
		values[selected[i]] = node.eval(E, x, y, args)
	}
	return values
}

func (E Expression) Eval(x, y float64) Color {
	values := E.EvalNodes(x, y, E.R, E.G, E.B)
	return Color{values[E.R], values[E.G], values[E.B]}
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

func (E Expression) RenderPixel(xlow, ylow, xhigh, yhigh float64, samples int) Color {
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
		c.Add(E.Eval(S[i], S[i+1]))
	}
	return c.Divide(float64(samples))
}

func (E Expression) Render(size, samples int) image.Image {
	var wg sync.WaitGroup
	img := NewImage(size, size)
	wg.Add(size)
	for i := 0; i < size; i++ {
		go func(i int) {
			for j := 0; j < size; j++ {
				xlow := float64(i) / float64(size)
				xhigh := float64(i+1) / float64(size)
				ylow := float64(j) / float64(size)
				yhigh := float64(j+1) / float64(size)
				c := E.RenderPixel(xlow, ylow, xhigh, yhigh, samples)
				img.px[i][j] = color.RGBA{
					uint8(_map(c.R) * 255.0),
					uint8(_map(c.G) * 255.0),
					uint8(_map(c.B) * 255.0),
					255,
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	return img
}

func (E Expression) String() string {
	s := "["
	for i, node := range E.Nodes {
		if i > 0 {
			s += "; "
		}
		colon := ""
		if E.R == i {
			s += "r"
			colon = ": "
		}
		if E.G == i {
			s += "g"
			colon = ": "
		}
		if E.B == i {
			s += "b"
			colon = ": "
		}
		s += colon
		s += node.Name()
		if node.Op == Const {
			s += fmt.Sprintf(" %g", node.Const)
		} else {
			for _, arg := range node.Args {
				s += fmt.Sprintf(" %d", arg)
			}
		}
	}
	s += "]"
	return s
}

func Read(s string) (expr Expression, err error) {
	if len(s) == 0 {
		return
	}
	s = strings.TrimSpace(s)
	if s[0] != '[' {
		err = fmt.Errorf("Expression does not start with '['")
		return
	}
	if s[len(s)-1] != ']' {
		err = fmt.Errorf("Expressions does not end with '['")
		return
	}
	s = s[1 : len(s)-1]

	for i, snod := range strings.Split(s, ";") {
		parts := strings.Split(snod, ":")
		switch len(parts) {
		case 1:
			snod = parts[0]
		case 2:
			for _, c := range strings.TrimSpace(parts[0]) {
				switch c {
				case 'r':
					expr.R = i
				case 'g':
					expr.G = i
				case 'b':
					expr.B = i
				}
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
		if op == "=" {
			// A constant
			node = &Node{Op: Const}
			n, _ := fmt.Fscanf(rnod, "%f", &node.Const)
			if n != 1 {
				err = fmt.Errorf("Error in node %d: cannot read constant", i)
				return
			}
		} else {
			id, ok := Ids[op]
			if !ok {
				err = fmt.Errorf("Error in node %d: operation '%s' unknown", i, op)
				return
			}
			node = &Node{Op: id}
			for {
				var arg int
				n, err := fmt.Fscanf(rnod, "%d", &arg)
				if n != 1 || err != nil {
					break
				}
				node.Args = append(node.Args, arg)
			}
			info := OperatorInfo[node.Op]
			if info.Nargs != len(node.Args) {
				err = fmt.Errorf("Error in node %d: '%s' should have %d args", i, op, info.Nargs)
				return
			}
		}
		expr.Nodes = append(expr.Nodes, node)
	}
	expr.TopologicalSort()	
	return expr.TreeShake(expr.R, expr.G, expr.B), nil
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
