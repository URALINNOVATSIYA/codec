package codec

import (
	"reflect"
	"testing"
)

type node struct {
	childId, parentId int
}

func TestGraph_Add(t *testing.T) {
	items := []struct {
		nodes   []node
		childs  map[int][]int
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
		// #2
		{
			[]node{
				{1, 0},
				{2, 1}, {12, 1},
				{3, 2}, {4, 2}, {5, 2},
				{4, 5}, {6, 5}, {7, 5},
				{7, 6}, {8, 6}, {9, 6}, {11, 6},
				{10, 8},
				{10, 9},
				{4, 11}, {5, 11},
				{13, 12}, {14, 12},
				{7, 13},
			},
			map[int][]int{
				0:  {1},
				1:  {2, 12},
				2:  {3, 4, 5},
				5:  {4, 6, 7},
				6:  {7, 8, 9, 11},
				8:  {10},
				9:  {10},
				11: {4, 5},
				12: {13, 14},
				13: {7},
			},
			map[int][]int{
				1:  {0},
				2:  {1},
				3:  {2},
				4:  {2, 5, 11},
				5:  {2, 11},
				6:  {5},
				7:  {5, 6, 13},
				8:  {6},
				9:  {6},
				10: {8, 9},
				11: {6},
				12: {1},
				13: {12},
				14: {12},
			},
		},
	}
	for i, item := range items {
		graph := newGraph()
		for _, node := range item.nodes {
			graph.addNode(node.childId, node.parentId)
		}
		if !reflect.DeepEqual(graph.childs, item.childs) {
			t.Errorf("Test #%d: actual graph children structure is incorrect: %#v", i+1, graph.childs)
		}
		if !reflect.DeepEqual(graph.prnts, item.parents) {
			t.Errorf("Test #%d: actual graph parents structure is incorrect: %#v", i+1, graph.prnts)
		}
	}
}

func TestGraph_Renumber(t *testing.T) {
	items := []struct {
		nodes                      []node
		childs                     map[int][]int
		parents                    map[int][]int
		currentNodeId, breakNodeId int
	}{
		// #1
		{
			[]node{
				{1, 0},
				{2, 1}, {6, 1},
				{3, 2},
				{4, 3},
				{5, 4},
			},
			map[int][]int{
				0: {1},
				1: {2, 4},
				2: {3},
				3: {4},
				4: {5},
				5: {6},
			},
			map[int][]int{
				1: {0},
				2: {1},
				3: {2},
				4: {1, 3},
				5: {4},
				6: {5},
			},
			6, 4,
		},
		// #2
		{
			[]node{
				{1, 0},
				{2, 1}, {12, 1},
				{3, 2}, {4, 2}, {5, 2},
				{4, 5}, {6, 5}, {7, 5},
				{7, 6}, {8, 6}, {9, 6}, {11, 6},
				{2, 8}, {10, 8},
				{10, 9},
				{4, 11}, {5, 11},
				{13, 12}, {14, 12},
				{7, 13},
			},
			map[int][]int{
				0:  {1},
				1:  {2, 5},
				2:  {3, 4, 5},
				5:  {6, 7},
				6:  {10},
				7:  {8},
				8:  {4, 9, 10},
				9:  {10, 11, 12, 14},
				11: {2, 13},
				12: {13},
				14: {4, 8},
			},
			map[int][]int{
				1:  {0},
				2:  {1, 11},
				3:  {2},
				4:  {2, 8, 14},
				5:  {1},
				6:  {5},
				7:  {5, 2, 14},
				8:  {7},
				9:  {8},
				10: {8, 9, 6},
				11: {9},
				12: {9},
				13: {11, 12},
				14: {9},
			},
			14, 5,
		},
	}
	for i, item := range items {
		graph := newGraph()
		for _, node := range item.nodes {
			graph.addNode(node.childId, node.parentId)
		}
		graph.renumber(item.currentNodeId, item.breakNodeId)
		if !reflect.DeepEqual(graph.childs, item.childs) {
			t.Errorf("Test #%d: actual graph children structure is incorrect: %#v", i+1, graph.childs)
		}
		if !reflect.DeepEqual(graph.prnts, item.parents) {
			t.Errorf("Test #%d: actual graph parents structure is incorrect: %#v", i+1, graph.prnts)
		}
	}
}
