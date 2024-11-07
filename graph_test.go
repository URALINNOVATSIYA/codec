package codec

import (
	"reflect"
	"testing"
)

func TestGraph_add(t *testing.T) {
	type node struct {
		childId, parentId int
	}
	items := []struct{
		nodes []node
		childs map[int][]int
		parents map[int][]int
	}{
		// #1
		{
			[]node{
				{1, 0}, {2, 0}, {3, 0},
				{4, 1}, {5, 1},
				{6, 3},
				{7, 6},
			},
			map[int][]int{
				0: {1, 2, 3},
				1: {4, 5},
				3: {6},
				6: {7},
			},
			map[int][]int{
				1: {0},
				2: {0},
				3: {0},
				4: {1},
				5: {1},
				6: {3},
				7: {6},
			},
		},
	}
	for i, item := range items {
		graph := newGraph()
		for _, node := range item.nodes {
			graph.add(node.childId, node.parentId)
		}
		if !reflect.DeepEqual(graph.childs, item.childs) {
			t.Errorf("Test #%d: expected graph children structure is incorrect: %#v", i, graph.childs)
		}
		if !reflect.DeepEqual(graph.prnts, item.parents) {
			t.Errorf("Test #%d: expected graph parents structure is incorrect: %#v", i, graph.prnts)
		}
	}
}