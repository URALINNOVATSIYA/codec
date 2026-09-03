package codec

import (
	"maps"
	"reflect"
)

type tmpValue struct {
	id int
	v  Value
}

type Graph struct {
	children   map[int][]int
	parents    map[int][]int
	vmap       map[int]struct{}
	lmap       map[int]struct{}
	values     map[int]Value
	tmpValues  map[int]tmpValue
	containers map[Addr]int
	addresses  map[Addr]int
}

func NewGraph() *Graph {
	return &Graph{
		children:   make(map[int][]int),
		parents:    make(map[int][]int),
		vmap:       make(map[int]struct{}),
		lmap:       make(map[int]struct{}),
		values:     make(map[int]Value),
		containers: make(map[Addr]int),
		addresses:  make(map[Addr]int),
	}
}

func (g *Graph) Children(parentId int) []int {
	return g.children[parentId]
}

func (g *Graph) ContainerAt(addr Addr) (int, bool) {
	nodeId, exists := g.containers[addr]
	return nodeId, exists
}

func (g *Graph) NodeAt(addr Addr) (int, bool) {
	nodeId, exists := g.addresses[addr]
	return nodeId, exists
}

func (g *Graph) GetNodeValue(nodeId int) reflect.Value {
	return g.values[nodeId].V
}

func (g *Graph) GetNode(nodeId int) Value {
	return g.values[nodeId]
}

func (g *Graph) SetNode(nodeId int, value Value) {
	g.values[nodeId] = value
	if value.Addr.IsValid() {
		g.addresses[value.Addr] = nodeId
	}
	if value.ContainerAddr.IsValid() {
		g.containers[value.ContainerAddr] = nodeId
	}
}

func (g *Graph) UpdateNodeValue(nodeId int, oldValue, newValue Value) {
	v := &oldValue
	if newValue.V.IsValid() {
		v.V = newValue.V
	}
	if newValue.Addr.IsValid() {
		if v.Addr.IsValid() {
			delete(g.addresses, v.Addr)
		}
		v.Addr = newValue.Addr
		g.addresses[v.Addr] = nodeId
	}
	if newValue.ContainerAddr.IsValid() {
		if v.ContainerAddr.IsValid() {
			delete(g.containers, v.ContainerAddr)
		}
		v.ContainerAddr = newValue.ContainerAddr
		g.containers[v.ContainerAddr] = nodeId
	}
	g.values[nodeId] = *v
}

func (g *Graph) AddNode(nodeId, parentId int) {
	g.children[parentId] = append(g.children[parentId], nodeId)
	g.parents[nodeId] = append(g.parents[nodeId], parentId)
}

func (g *Graph) AddNodeWithValue(nodeId, parentId int, value Value) {
	g.AddNode(nodeId, parentId)
	g.SetNode(nodeId, value)
}

func (g *Graph) IsVisited(nodeId int) bool {
	_, exists := g.vmap[nodeId]
	return exists
}

func (g *Graph) Visit(nodeId int) {
	g.vmap[nodeId] = struct{}{}
}

func (g *Graph) IsLoop(nodeId int) bool {
	_, exists := g.lmap[nodeId]
	return exists
}

func (g *Graph) Loop(nodeId int) {
	g.lmap[nodeId] = struct{}{}
}

func (g *Graph) Fix(currentNodeId, collisionNodeId int) {
	maxNodeId := g.findMaxNodeId(collisionNodeId, collisionNodeId, collisionNodeId, make(map[int]struct{}))

	if maxNodeId == currentNodeId {
		for _, parentId := range g.parents[collisionNodeId] {
			for i := range g.children[parentId] {
				g.children[parentId][i] = currentNodeId
				g.parents[currentNodeId] = append(g.parents[currentNodeId], parentId)
			}
		}
		g.children[currentNodeId] = []int{collisionNodeId}
		g.parents[collisionNodeId] = []int{currentNodeId}
		return
	}

	inc := currentNodeId - maxNodeId
	dec := maxNodeId - collisionNodeId + 1
	g.tmpValues = make(map[int]tmpValue)

	g.fixBorderNodes(collisionNodeId, maxNodeId, inc, dec)
	parents := g.fixParents(collisionNodeId, currentNodeId, maxNodeId, inc, dec)
	children := g.fixChildren(collisionNodeId, currentNodeId, maxNodeId, inc, dec, parents)

	currentNodeId -= dec
	collisionNodeId += inc
	parents[currentNodeId] = append(parents[currentNodeId], parents[collisionNodeId]...)
	parents[collisionNodeId] = []int{currentNodeId}
	children[currentNodeId] = []int{collisionNodeId}

	g.children = g.mergeNodes(g.children, children)
	g.parents = g.mergeNodes(g.parents, parents)

	g.fixValues()
}

func (g *Graph) findMaxNodeId(parentNodeId, minNodeId, maxNodeId int, vmap map[int]struct{}) int {
	for _, nodeId := range g.children[parentNodeId] {
		if nodeId <= minNodeId {
			continue
		}
		if _, visited := vmap[nodeId]; visited {
			continue
		}
		vmap[nodeId] = struct{}{}
		if nodeId > maxNodeId {
			maxNodeId = nodeId
		}
		maxNodeId = g.findMaxNodeId(nodeId, minNodeId, maxNodeId, vmap)
	}
	return maxNodeId
}

func (g *Graph) fixBorderNodes(startNodeId, turnNodeId, inc, dec int) {
	borderParents := append(g.parents[turnNodeId+1], g.parents[startNodeId]...)
	for _, parentId := range borderParents {
		if parentId >= startNodeId {
			continue
		}
		childs := g.children[parentId]
		for i, childId := range childs {
			if childId >= startNodeId {
				childs[i] = g.renumberNodeId(childId, turnNodeId, inc-1, dec)
			}
		}
	}
}

func (g *Graph) fixParents(startNodeId, endNodeId, turnNodeId, inc, dec int) map[int][]int {
	parents := make(map[int][]int)
	for nodeId := startNodeId; nodeId <= endNodeId; nodeId++ {
		elems := g.parents[nodeId]
		if len(elems) == 0 {
			continue
		}
		for i, id := range elems {
			if id < startNodeId {
				elems[i] = id
			} else {
				elems[i] = g.renumberNodeId(id, turnNodeId, inc, dec)
			}
		}
		parents[g.renumberNodeId(nodeId, turnNodeId, inc, dec)] = elems
		delete(g.parents, nodeId)
	}
	return parents
}

func (g *Graph) fixChildren(startNodeId, endNodeId, turnNodeId, inc, dec int, prnts map[int][]int) map[int][]int {
	childs := make(map[int][]int)
	for nodeId := startNodeId; nodeId <= endNodeId; nodeId++ {
		elems := g.children[nodeId]
		if len(elems) == 0 {
			continue
		}
		for i, id := range elems {
			if id > startNodeId {
				elems[i] = g.renumberNodeId(id, turnNodeId, inc, dec)
				continue
			}
			if id == startNodeId {
				elems[i] = g.renumberNodeId(id, turnNodeId, inc-1, dec)
				continue
			}
			elems[i] = id
			parents := g.parents[id]
			if parents == nil {
				continue
			}
			for j, parentId := range parents {
				if parentId >= startNodeId {
					parents[j] = g.renumberNodeId(parentId, turnNodeId, inc, dec)
				}
			}
			prnts[id] = parents
			delete(g.parents, id)
		}
		childs[g.renumberNodeId(nodeId, turnNodeId, inc, dec)] = elems
		delete(g.children, nodeId)
	}
	return childs
}

func (g *Graph) renumberNodeId(nodeId, turnNodeId, inc, dec int) int {
	id := nodeId
	if nodeId <= turnNodeId {
		id += inc
	} else {
		id -= dec
	}
	if v, exists := g.values[nodeId]; exists {
		g.tmpValues[id] = tmpValue{id: nodeId, v: v}
	}
	return id
}

func (g *Graph) mergeNodes(nodes1, nodes2 map[int][]int) map[int][]int {
	if len(nodes2) > len(nodes1) {
		nodes1, nodes2 = nodes2, nodes1
	}
	maps.Copy(nodes1, nodes2)
	return nodes1
}

func (g *Graph) fixValues() {
	visited := make([]int, 0, len(g.tmpValues))
	loops := make([]int, 0, len(g.tmpValues)>>1)
	for nodeId, v := range g.tmpValues {
		value := v.v
		g.values[nodeId] = value
		if value.Addr.IsValid() {
			g.addresses[value.Addr] = nodeId
		}
		if value.ContainerAddr.IsValid() {
			g.containers[value.ContainerAddr] = nodeId
		}
		if _, exists := g.vmap[v.id]; exists {
			visited = append(visited, nodeId)
			delete(g.vmap, v.id)
		}
		if _, exists := g.lmap[v.id]; exists {
			loops = append(loops, nodeId)
			delete(g.lmap, v.id)
		}
	}
	for _, nodeId := range visited {
		g.vmap[nodeId] = struct{}{}
	}
	for _, nodeId := range loops {
		g.lmap[nodeId] = struct{}{}
	}
	g.tmpValues = nil
}
