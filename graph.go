package codec

type graph struct {
	childs map[int][]int
	prnts map[int][]int
	vmap map[int]struct{}
}

func newGraph() *graph {
	return &graph{
		childs: make(map[int][]int),
		prnts: make(map[int][]int),
		vmap: make(map[int]struct{}),
	}
}

func (g *graph) add(childId, parentId int) {
	g.childs[parentId] = append(g.childs[parentId], childId)
	g.prnts[childId] = append(g.prnts[childId], parentId)
}

func (g *graph) nodes() map[int][]int {
	return g.childs
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
	g.renumberBorderNodes(maxNodeId+1, breakNodeId, maxNodeId, inc, dec, g.prnts, g.childs)
	prnts := g.renumberParents(breakNodeId, currentNodeId, maxNodeId, inc, dec)
	childs := g.renumberChilds(breakNodeId, currentNodeId, maxNodeId, inc, dec, prnts)
	
	currentNodeId -= dec
	breakNodeId += inc
	prnts[currentNodeId] = append(prnts[currentNodeId], prnts[breakNodeId]...)
	prnts[breakNodeId] = []int{currentNodeId}
	childs[currentNodeId] = []int{breakNodeId}

	g.merge(g.childs, childs)
	g.merge(g.prnts, prnts)
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

func (g *graph) renumberBorderNodes(borderNodeId, startNodeId, turnNodeId, inc, dec int, childs, prnts map[int][]int) {
	for _, childId := range childs[borderNodeId] {
		if childId >= startNodeId {
			continue
		}
		elems := prnts[childId]
		for i, nodeId := range elems {
			if nodeId >= startNodeId {
				elems[i] = g.renumberNodeId(nodeId, turnNodeId, inc, dec)
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
	if nodeId <= turnNodeId {
		return nodeId + inc
	}
	return nodeId - dec
}

func (g *graph) merge(nodes1, nodes2 map[int][]int) {
	for nodeId, nodes := range nodes2 {
		nodes1[nodeId] = nodes
	}
}