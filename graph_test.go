package codec

import (
	"reflect"
	"testing"
)

type graphNode struct {
	nodeId   int
	parentId int
}

func TestGraph_Fix(t *testing.T) {
	items := []struct {
		nodes           []graphNode
		children        map[int][]int
		parents         map[int][]int
		currentNodeId   int
		collisionNodeId int
	}{
		// #1 (see graph 1)
		{
			[]graphNode{
				{1, 0},
				{2, 1},
				{3, 2},
				{4, 3},
			},
			map[int][]int{
				0: {4},
				1: {2},
				2: {3},
				3: {4},
				4: {1},
			},
			map[int][]int{
				1: {4},
				2: {1},
				3: {2},
				4: {3, 0},
			},
			4,
			1,
		},
		// #2 (see graph 2)
		{
			[]graphNode{
				{1, 0},
				{2, 1},
				{3, 2},
				{4, 3},
				{5, 4},
				{6, 5},
				{7, 6},
				{8, 1},
			},
			map[int][]int{
				0: {1},
				1: {2, 6},
				2: {3},
				3: {4},
				4: {5},
				5: {6},
				6: {7},
				7: {8},
			},
			map[int][]int{
				1: {0},
				2: {1},
				3: {2},
				4: {3},
				5: {4},
				6: {1, 5},
				7: {6},
				8: {7},
			},
			8,
			6,
		},
	}
	for i, item := range items {
		graph := NewGraph()
		for _, node := range item.nodes {
			graph.AddNode(node.nodeId, node.parentId)
		}
		graph.Fix(item.currentNodeId, item.collisionNodeId)
		if !reflect.DeepEqual(graph.children, item.children) {
			t.Errorf("Test #%d: wrong children: %v", i+1, graph.children)
		}
		if !reflect.DeepEqual(graph.parents, item.parents) {
			t.Errorf("Test #%d: wrong parents: %v", i+1, graph.parents)
		}
		/*if !reflect.DeepEqual(graph.values, values) {
			t.Errorf("Test #%d: wrong values: %v", i+1, graph.parents)
		}*/
	}
}
