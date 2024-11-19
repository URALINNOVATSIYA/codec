package codec

import (
	"reflect"
	"unsafe"
)

type valueAddr struct {
	ptr      unsafe.Pointer
	typeName string
}

type nodeValue struct {
	v    reflect.Value
	addr valueAddr
	cntr valueAddr
}

func (a valueAddr) isValid() bool {
	return a.ptr != nil
}

type graph struct {
	childs map[int][]int
	prnts  map[int][]int
	vmap   map[int]struct{}
	values map[int]nodeValue
	tvals  map[int]nodeValue
	addrs  map[valueAddr]int
	cntrs  map[valueAddr]int
}

func newGraph() *graph {
	return &graph{
		childs: make(map[int][]int),
		prnts:  make(map[int][]int),
		vmap:   make(map[int]struct{}),
		values: make(map[int]nodeValue),
		addrs:  make(map[valueAddr]int),
		cntrs:  make(map[valueAddr]int),
	}
}

func (g *graph) addNodeWithValue(childId, parentId int, value nodeValue) {
	g.addNode(childId, parentId)
	g.addNodeValue(childId, value)
}

func (g *graph) addNode(childId, parentId int) {
	g.childs[parentId] = append(g.childs[parentId], childId)
	g.prnts[childId] = append(g.prnts[childId], parentId)
}

func (g *graph) addNodeValue(nodeId int, value nodeValue) {
	g.values[nodeId] = value
	if value.addr.isValid() {
		g.addrs[value.addr] = nodeId
	}
	if value.cntr.isValid() {
		g.cntrs[value.cntr] = nodeId
	}
}

func (g *graph) updateNodeValue(nodeId int, oldValue, newValue nodeValue) {
	v := &oldValue
	if newValue.v.IsValid() {
		v.v = newValue.v
	}
	if newValue.addr.isValid() {
		if v.addr.isValid() {
			delete(g.addrs, v.addr)
		}
		v.addr = newValue.addr
		g.addrs[v.addr] = nodeId
	}
	if newValue.cntr.isValid() {
		if v.cntr.isValid() {
			delete(g.cntrs, v.cntr)
		}
		v.cntr = newValue.cntr
		g.cntrs[v.cntr] = nodeId
	}
	g.values[nodeId] = *v
}

func (g *graph) get(nodeId int) reflect.Value {
	return g.values[nodeId].v
}

func (g *graph) nodeValue(nodeId int) nodeValue {
	return g.values[nodeId]
}

func (g *graph) children(parentId int) []int {
	return g.childs[parentId]
}

func (g *graph) nodeAt(addr valueAddr) (int, bool) {
	nodeId, exists := g.addrs[addr]
	return nodeId, exists
}

func (g *graph) containerNodeAt(addr valueAddr) (int, bool) {
	nodeId, exists := g.cntrs[addr]
	return nodeId, exists
}

func (g *graph) isVisited(nodeId int) bool {
	_, exists := g.vmap[nodeId]
	return exists
}

func (g *graph) visit(nodeId int) {
	g.vmap[nodeId] = struct{}{}
}

func (g *graph) renumber(currentNodeId, breakNodeId int) {
	maxNodeId := g.findMaxNodeId(breakNodeId, breakNodeId, breakNodeId, make(map[int]struct{}))

	inc := currentNodeId - maxNodeId
	dec := maxNodeId - breakNodeId + 1
	g.tvals = make(map[int]nodeValue)

	g.renumberBorderNodes(breakNodeId, maxNodeId, inc, dec)
	prnts := g.renumberParents(breakNodeId, currentNodeId, maxNodeId, inc, dec)
	childs := g.renumberChilds(breakNodeId, currentNodeId, maxNodeId, inc, dec, prnts)

	currentNodeId -= dec
	breakNodeId += inc
	prnts[currentNodeId] = append(prnts[currentNodeId], prnts[breakNodeId]...)
	prnts[breakNodeId] = []int{currentNodeId}
	childs[currentNodeId] = []int{breakNodeId}

	g.childs = g.merge(g.childs, childs)
	g.prnts = g.merge(g.prnts, prnts)
	g.restoreMeta()
}

func (g *graph) findMaxNodeId(parentNodeId, minNodeId, maxNodeId int, vmap map[int]struct{}) int {
	for _, nodeId := range g.childs[parentNodeId] {
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

func (g *graph) renumberBorderNodes(startNodeId, turnNodeId, inc, dec int) {
	borderParents := append(g.prnts[turnNodeId+1], g.prnts[startNodeId]...)
	for _, parentId := range borderParents {
		if parentId >= startNodeId {
			continue
		}
		childs := g.childs[parentId]
		for i, childId := range childs {
			if childId >= startNodeId {
				childs[i] = g.renumberNodeId(childId, turnNodeId, inc-1, dec)
			}
		}
	}
}

func (g *graph) renumberParents(startNodeId, endNodeId, turnNodeId, inc, dec int) map[int][]int {
	prnts := make(map[int][]int)
	for nodeId := startNodeId; nodeId <= endNodeId; nodeId++ {
		elems := g.prnts[nodeId]
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
		prnts[g.renumberNodeId(nodeId, turnNodeId, inc, dec)] = elems
		delete(g.prnts, nodeId)
	}
	return prnts
}

func (g *graph) renumberChilds(startNodeId, endNodeId, turnNodeId, inc, dec int, prnts map[int][]int) map[int][]int {
	childs := make(map[int][]int)
	for nodeId := startNodeId; nodeId <= endNodeId; nodeId++ {
		elems := g.childs[nodeId]
		if len(elems) == 0 {
			continue
		}
		for i, id := range elems {
			if id >= startNodeId {
				elems[i] = g.renumberNodeId(id, turnNodeId, inc, dec)
				continue
			}
			elems[i] = id
			parents := g.prnts[id]
			if parents == nil {
				continue
			}
			for j, parentId := range parents {
				if parentId >= startNodeId {
					parents[j] = g.renumberNodeId(parentId, turnNodeId, inc, dec)
				}
			}
			prnts[id] = parents
			delete(g.prnts, id)
		}
		childs[g.renumberNodeId(nodeId, turnNodeId, inc, dec)] = elems
		delete(g.childs, nodeId)
	}
	return childs
}

func (g *graph) renumberNodeId(nodeId, turnNodeId, inc, dec int) int {
	id := nodeId
	if nodeId <= turnNodeId {
		id += inc
	} else {
		id -= dec
	}
	g.rebindMeta(nodeId, id)
	return id
}

func (g *graph) rebindMeta(oldNodeId, newNodeId int) {
	if v, exists := g.values[oldNodeId]; exists {
		g.tvals[newNodeId] = v
	}
}

func (g *graph) restoreMeta() {
	for nodeId, v := range g.tvals {
		g.values[nodeId] = v
		if v.addr.isValid() {
			g.addrs[v.addr] = nodeId
		}
		if v.cntr.isValid() {
			g.cntrs[v.cntr] = nodeId
		}
	}
	g.tvals = nil
}

func (g *graph) merge(nodes1, nodes2 map[int][]int) map[int][]int {
	if len(nodes2) > len(nodes1) {
		nodes1, nodes2 = nodes2, nodes1
	}
	for nodeId, nodes := range nodes2 {
		nodes1[nodeId] = nodes
	}
	return nodes1
}
