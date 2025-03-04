package apihandler

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestGraphTemplate(t *testing.T) {
	var graphs []GraphMeta
	graphs = append(graphs, GraphMeta{
		ID:   "1",
		Name: "test",
		Href: "http://localhost:8080/graphs/1",
	})

	var res bytes.Buffer
	tmpl := template.Must(template.New("listGraphs").Parse(listGraphs))
	err := tmpl.Execute(&res, graphs)

	assert.NoError(t, err)
	assert.NotEmpty(t, res.String())
}
