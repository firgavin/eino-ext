package apihandler

import (
	"fmt"
	"net/http"
	"reflect"
	"text/template"

	"github.com/firgavin/eino-devops/internal/service"
	"github.com/gorilla/mux"
)

const debugVis = "/debug/v2"

const graphTpl = `
<!DOCTYPE html>
<html>
<head>
    <title>Workflow Visualization</title>
    <style>
        #network-container {
            width: 100%;
            height: 100vh;
            border: 1px solid #ccc;
        }
    </style>
    <script type="text/javascript" src="https://unpkg.com/vis-network@9.1.2/standalone/umd/vis-network.min.js"></script>
</head>
<body>
    <div id="network-container"></div>

    <script type="text/javascript">
        // 颜色回退函数必须在前
        function getGroupColor(type) {
            var colorMap = {
                'start': {background:'#4CAF50', border:'#388E3C'},
                'end': {background:'#F44336', border:'#D32F2F'},
                'parallel': {background:'#2196F3', border:'#1976D2'},
                'branch': {background:'#FFC107', border:'#FFA000'},
                'Lambda': {background:'#9C27B0', border:'#7B1FA2'},
                'Passthrough': {background:'#00BCD4', border:'#0097A7'}
            };
            return colorMap[type] || {background:'#9E9E9E', border:'#616161'};
        }

        // 动态生成的节点数据
        var nodes = new vis.DataSet([
            {{range $index, $node := .Nodes}}{
                id: "{{$node.Key | js}}",
                label: {{$node.Label | printf "%q"}},
                group: "{{$node.Type | js}}",
                shape: 'box',
                font: { size: 14 },
                color: getGroupColor("{{$node.Type | js}}")
            }{{if not (last $index $.Nodes)}},{{end}}
            {{end}}
        ]);

        // 动态生成的边数据
        var edges = new vis.DataSet([
            {{range $index, $edge := .Edges}}{
                from: "{{$edge.From | js}}",
                to: "{{$edge.To | js}}"
            }{{if not (last $index $.Edges)}},{{end}}
            {{end}}
        ]);

        // 初始化网络
        var container = document.getElementById('network-container');
        var data = { nodes: nodes, edges: edges };

		var options = {
			nodes: {
				fixed: {
					x: false,
					y: false
				},
				margin: 15,
				shape: 'box',
				widthConstraint: {
					maximum: 150
				}
			},
			edges: {
				arrows: 'to',
				smooth: {
					type: 'cubicBezier',
					forceDirection: 'horizontal'
				}
			},
			layout: {
				hierarchical: {
					enabled: true,       // 必须启用层级布局
					direction: "LR",     // Left to Right
					sortMethod: 'directed',
					nodeSpacing: 120,    // 水平节点间距
					levelSeparation: 200,// 垂直层级间距
					shakeTowards: 'roots'
				}
			},
			physics: {
				hierarchicalRepulsion: {
					nodeDistance: 200,
					centralGravity: 0,
					springLength: 200,
					springConstant: 0.01,
					damping: 0.09
				},
				solver: 'hierarchicalRepulsion'
			},
			interaction: {
				dragNodes: true,
				dragView: true,
				zoomView: true
			}
		};


        var network = new vis.Network(container, data, options);
		
		network.on("stabilizationIterationsDone", function() {
			network.fit({
				animation: {
					duration: 800,
					easingFunction: 'easeInOutQuad'
				}
			});
			
			// 打印层级结构调试信息
			console.log("Hierarchy levels:", 
				network.getPositions().map(pos => pos.y)
					.filter((v, i, a) => a.indexOf(v) === i)
					.sort()
			);
		});
    </script>
</body>
</html>
`

type SimpleGraph struct {
	Nodes []struct {
		Key   string
		Label string
		Type  string
	}
	Edges []struct {
		From string
		To   string
	}
}

func DrawGraph(res http.ResponseWriter, req *http.Request) {
	graphID := getPathParam(req, "graph_id")
	if len(graphID) == 0 {
		newHTTPResp(newBizError(http.StatusBadRequest, fmt.Errorf("graph_name is empty")), newBaseResp(http.StatusBadRequest, "")).doResp(res)
		return
	}

	canvasInfo, exist := service.ContainerSVC.GetCanvas(graphID)
	if !exist {
		newCanvasInfo, err := service.ContainerSVC.CreateCanvas(graphID)
		if err != nil {
			newHTTPResp(newBizError(http.StatusBadRequest, err), newBaseResp(http.StatusBadRequest, "")).doResp(res)
			return
		}
		canvasInfo = newCanvasInfo
	}

	allNodes, allEdges := flattenGraph(canvasInfo.GraphSchema)

	var tplData SimpleGraph
	for _, node := range allNodes {
		label := node.Name
		if label == "" {
			if node.ComponentSchema.Name != "" {
				label = node.ComponentSchema.Name
			} else {
				label = node.Key
			}
		}

		tplData.Nodes = append(tplData.Nodes, struct {
			Key   string
			Label string
			Type  string
		}{
			Key:   node.Key,
			Label: label,
			Type:  string(node.Type),
		})
	}
	for _, edge := range allEdges {
		tplData.Edges = append(tplData.Edges, struct {
			From string
			To   string
		}{
			From: edge.SourceNodeKey,
			To:   edge.TargetNodeKey,
		})
	}

	tmpl := template.Must(template.New("visualization").Funcs(template.FuncMap{
		"printf": fmt.Sprintf,
		"last": func(index int, slice interface{}) bool {
			v := reflect.ValueOf(slice)
			return index == v.Len()-1
		},
	}).Parse(graphTpl))

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(res, tplData)
}

var listGraphs = ` 
<html>
<head>
	<title>Graphs</title>
	<style>a {text-decoration: none;}</style>
</head>
<body>
	<p>
		<a href="/graphs">&lt;&lt; Graphs</a>
	</p>
	<table>
		<th>
			<tr>
				<td>Graphs</td>
			</tr>
		</th>
		<tbody>{{ range . }}
			<tr>
				<td><a href="{{ .Href }}">{{ .ID }}/{{ .Name }}</a></td>
			</tr>
		{{ end }}</tbody>
	</table>
</body>
</html>
`

type GraphMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Href string `json:"href"`
}

func ShowGraphs(res http.ResponseWriter, req *http.Request) {
	graphNameToID := service.ContainerSVC.ListGraphs()
	graphs := make([]GraphMeta, 0, len(graphNameToID))
	for name, id := range graphNameToID {
		graphs = append(graphs, GraphMeta{
			ID:   id,
			Name: name,
			Href: fmt.Sprintf("%s/graphs/%s", debugVis, id),
		})
	}

	tmpl := template.Must(template.New("listGraphs").Parse(listGraphs))
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(res, graphs)
}

func registerVisualizeRoutes(r *mux.Router) {
	r.Use(recoverMiddleware, corsMiddleware)
	debugR := r.PathPrefix(debugVis).Subrouter()
	debugR.Path("/graphs").HandlerFunc(ShowGraphs).Methods(http.MethodGet)
	debugR.Path("/graphs/{graph_id}").HandlerFunc(DrawGraph).Methods(http.MethodGet)
}
