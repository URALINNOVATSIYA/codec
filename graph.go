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

func (g *graph) children(parentId int) []int {
	return g.childs[parentId]
}

func (g *graph) parents(childId int) []int {
	return g.prnts[childId]
}

func (g *graph) isVisited(nodeId int) bool {
	_, exists := g.vmap[nodeId]
	return exists
}

func (g *graph) visit(nodeId int) {
	g.vmap[nodeId] = struct{}{}
}

func (g *graph) renumber(currentNodeId, breakNodeId int) {

}