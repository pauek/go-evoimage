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
	Const = iota
	Id
	X
	Y
	R
	T
	Sin
	Cos
	Sum
	Mult
	Lerp
	Inv
	Band
	Bw
	And
	Or
	Xor
	Not
	If
	Blur
)

const MAX_ARGS = 10

type OpInfo struct {
	Name  string
	Nargs int
	Neigh bool
}

var OperatorInfo = map[int]OpInfo{
	Const: {"=", 1, false},
	Id:    {"id", 1, false},
	X:     {"x", 0, false},
	Y:     {"y", 0, false},
	R:     {"r", 0, false},
	T:     {"t", 0, false},
	Cos:   {"cos", 1, false},
	Sin:   {"sin", 1, false},
	Sum:   {"+", 2, false},
	Mult:  {"*", 2, false},
	Lerp:  {"lerp", 3, false},
	Inv:   {"inv", 1, false},
	Band:  {"band", 1, false},
	Bw:    {"bw", 1, false},
	And:   {"and", 2, false},
	Or:    {"or", 2, false},
	Xor:   {"xor", 2, false},
	Not:   {"not", 1, false},
	If:    {"if", 3, false},
	Blur:  {"blur", 1, true},
}

var Ids = map[string]int{}

func init() {
	for id, info := range OperatorInfo {
		Ids[info.Name] = id
	}
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

func (N *Node) Name() string {
	return OperatorInfo[N.Op].Name
}

type Expression []*Node

type Color struct {
	R, G, B float64
}

type NodeFunc func(args []float64) float64

func (E *Expression) Size() int {
	return len(*E)
}

func (E *Expression) ForEach(f func(i int, n *Node)) {
	for i, node := range *E {
		f(i, node)
	}
}

// Ordenación topológica: los nodos estan puestos de tal manera
// que no hay dependencias hacia nodos de menor índice.

type Topological []*_Node

func (t Topological) Len() int           { return len(t) }
func (t Topological) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t Topological) Less(i, j int) bool { return t[i].Order > t[j].Order }

func (E Expression) TopologicalSort() {
	_E := make([]*_Node, len(E))
	for i := range E {
		_E[i] = &_Node{
			Node:  *E[i],
			Order: -1,
		}
	}
	changes := true
	for changes {
		changes = false
		for _, node := range _E {
			if node.Order >= 0 {
				continue
			}
			max_child_order := 0 // for no-args nodes
			for _, arg := range node.Args {
				ord := _E[arg].Order
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
	old_E := make([]*_Node, len(_E))
	for i := range _E {
		old_E[i] = _E[i]
	}
	sort.Sort(Topological(_E))
	for i := range _E {
		_E[i].NewPos = i
	}
	for i := range _E {
		for j := range _E[i].Args {
			iold := _E[i].Args[j]
			inew := old_E[iold].NewPos
			_E[i].Args[j] = inew
		}
		E[i] = &_E[i].Node
	}
}

func (E Expression) TreeShake(roots ...int) Expression {
	sz := len(E)
	if sz == 0 {
		return Expression{}
	}
	order := make([]int, sz+1)
	for i := range order {
		order[i] = -1
	}
	top := 0
	for _, root := range roots {
		order[top] = root
		top++
	}
	curr := 0
	var newE Expression
	for order[curr] != -1 {
		node := E[order[curr]]
		newnode := Node{
			Op:    node.Op,
			Args:  make([]int, len(node.Args)),
			Const: node.Const,
		}
		for j, arg := range node.Args {
			order[top] = arg
			newnode.Args[j] = top
			top++
		}
		newE = append(newE, &newnode)
		curr++
	}
	return newE
}

func (node *Node) eval(x, y float64, args []float64) float64 {
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
	default:
		panic("not implemented")
	}
}

const BLUR_SAMPLES = 3
const BLUR_RADIUS = 0.05

func (node *Node) evalNeigh(E Expression, x, y float64, args []float64) float64 {
	switch node.Op {
	case Blur:
		v := 0.0
		d := float64(BLUR_RADIUS) / float64(BLUR_SAMPLES)
		for i := 0; i < BLUR_SAMPLES; i++ {
			for j := 0; j < BLUR_SAMPLES; j++ {
				dx := float64(i) * d + d/2 * rand.Float64()
				dy := float64(j) * d + d/2 * rand.Float64()
				v += E.EvalNode(node.Args[0], x + dx, y + dy)
			}
		}
		return v / float64(BLUR_SAMPLES * BLUR_SAMPLES)
		
	default:
		panic("not implemented")
	}
}

func (E Expression) EvalNode(root int, x, y float64) float64 {
	// Select nodes that we will compute
	selected := make([]int, len(E))
	selected[0] = root
	top := 1
	for i := 0; i < top; i++ {
		node := E[selected[i]]
		for _, arg := range node.Args {
			selected[top] = arg
			top++
		}
	}
	values := make([]float64, len(E))
	args := make([]float64, MAX_ARGS)
	for i := top - 1; i >= 0; i-- {
		node := E[selected[i]]
		for j, arg := range node.Args {
			args[j] = values[arg]
		}
		if OperatorInfo[node.Op].Neigh {
			values[selected[i]] = node.evalNeigh(E, x, y, args)
		} else {
			values[selected[i]] = node.eval(x, y, args)
		}
	}
	return values[root]
}

func (E Expression) Eval(x, y float64) float64 {
	return E.EvalNode(0, x, y)
}

func Map(x float64) (y float64) {
	y = x
	if y > 1.0 {
		y = 1.0
	}
	if y < 0.0 {
		y = 0.0
	}
	return
}

func (E Expression) Render(size, samples int) image.Image {
	var wg sync.WaitGroup
	img := NewImage(size, size)
	wg.Add(size)
	for i := 0; i < size; i++ {
		go func(i int) {
			for j := 0; j < size; j++ {
				c := Color{0, 0, 0}
				if samples == 1 {
					x := float64(i) / float64(size-1)
					y := float64(size-1-j) / float64(size-1)
					v := E.Eval(x, y)
					c = Color{v, v, v}
				} else {
					for k := 0; k < samples; k++ {
						dx := .5 + .4*rand.Float64()
						dy := .5 + .4*rand.Float64()
						x := (float64(i) + dx) / float64(size)
						y := (float64(size-1-j) + dy) / float64(size)
						rgb := E.Eval(x, y)
						c.R += rgb
						c.G += rgb
						c.B += rgb
					}
					c.R /= float64(samples)
					c.G /= float64(samples)
					c.B /= float64(samples)
				}
				img.px[i][j] = color.RGBA{
					uint8(Map(c.R) * 255.0),
					uint8(Map(c.G) * 255.0),
					uint8(Map(c.B) * 255.0),
					255,
				}
			}
			wg.Done()
		} (i)
	}
	wg.Wait()
	return img
}

func (E Expression) String() string {
	s := "["
	for i, node := range E {
		if i > 0 {
			s += "; "
		}
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
		return nil, fmt.Errorf("Expression does not start with '['")
	}
	if s[len(s)-1] != ']' {
		return nil, fmt.Errorf("Expressions does not end with '['")
	}
	s = s[1 : len(s)-1]

	for i, snod := range strings.Split(s, ";") {
		rnod := strings.NewReader(snod)

		var op string
		n, err := fmt.Fscanf(rnod, "%s", &op)
		if n != 1 || err != nil {
			return nil, fmt.Errorf("Error in node %d: '%s'", i, snod)
		}

		var node *Node
		if op == "=" {
			// A constant
			node = &Node{Op: Const}
			n, _ := fmt.Fscanf(rnod, "%f", &node.Const)
			if n != 1 {
				return nil, fmt.Errorf("Error in node %d: cannot read constant", i)
			}
		} else {
			id, ok := Ids[op]
			if !ok {
				return nil, fmt.Errorf("Error in node %d: operation '%s' unknown", i, op)
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
				return nil, fmt.Errorf("Error in node %d: operation '%s' should have %d args",
					i, op, info.Nargs)
			}
		}
		expr = append(expr, node)
	}
	return expr.TreeShake(0), nil
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