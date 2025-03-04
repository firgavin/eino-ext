package apihandler

import (
	"fmt"

	devmodel "github.com/firgavin/eino-devops/model"
)

func flattenGraph(graph *devmodel.GraphSchema) (allNodes []*devmodel.Node, allEdges []*devmodel.Edge) {
	if graph == nil {
		return nil, nil
	}

	for _, node := range graph.Nodes {
		if node.GraphSchema != nil {
			subNodes, subEdges := processNestedGraph(node)
			allNodes = append(allNodes, subNodes...)
			allEdges = append(allEdges, subEdges...)
		} else {
			allNodes = append(allNodes, node)
		}
	}
	allEdges = append(allEdges, graph.Edges...)

	return deduplicate(allNodes, allEdges)
}

func processNestedGraph(parentNode *devmodel.Node) ([]*devmodel.Node, []*devmodel.Edge) {
	subGraph := parentNode.GraphSchema

	var startNode, endNode *devmodel.Node
	for _, n := range subGraph.Nodes {
		switch n.Type {
		case devmodel.NodeTypeOfStart:
			startNode = n
		case devmodel.NodeTypeOfEnd:
			endNode = n
		}
	}

	subNodes, subEdges := flattenGraph(subGraph)

	// 处理父节点的边连接
	modifiedEdges := make([]*devmodel.Edge, 0)
	for _, e := range parentNode.GraphSchema.Edges {
		// 替换入边：其他节点 -> 父节点 => 其他节点 -> 子图开始节点
		if e.TargetNodeKey == parentNode.Key {
			modifiedEdges = append(modifiedEdges, &devmodel.Edge{
				SourceNodeKey: e.SourceNodeKey,
				TargetNodeKey: startNode.Key,
			})
		}

		// 替换出边：父节点 -> 其他节点 => 子图结束节点 -> 其他节点
		if e.SourceNodeKey == parentNode.Key {
			modifiedEdges = append(modifiedEdges, &devmodel.Edge{
				SourceNodeKey: endNode.Key,
				TargetNodeKey: e.TargetNodeKey,
			})
		}
	}

	return append(subNodes, startNode, endNode),
		append(subEdges, modifiedEdges...)
}

func deduplicate(nodes []*devmodel.Node, edges []*devmodel.Edge) ([]*devmodel.Node, []*devmodel.Edge) {
	nodeMap := make(map[string]*devmodel.Node)
	edgeMap := make(map[string]*devmodel.Edge)

	for _, n := range nodes {
		nodeMap[n.Key] = n
	}
	for _, e := range edges {
		key := fmt.Sprintf("%s->%s", e.SourceNodeKey, e.TargetNodeKey)
		edgeMap[key] = e
	}

	uniqueNodes := make([]*devmodel.Node, 0, len(nodeMap))
	for _, v := range nodeMap {
		uniqueNodes = append(uniqueNodes, v)
	}
	uniqueEdges := make([]*devmodel.Edge, 0, len(edgeMap))
	for _, v := range edgeMap {
		uniqueEdges = append(uniqueEdges, v)
	}

	return uniqueNodes, uniqueEdges
}
