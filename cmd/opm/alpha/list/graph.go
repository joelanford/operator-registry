package list

import (
	"fmt"
	"os"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/internal/action"
	"github.com/operator-framework/operator-registry/internal/declcfg"
)

func newGraphCmd() *cobra.Command {
	logger := logrus.New()

	return &cobra.Command{
		Use:   "graph <directory> <packageName> <channelName>",
		Short: "Show graph for a package channel in an index",
		Long: `The "graph" command outputs a DAG based on the edges defined in the specified
index, package, and channel.

` + humanReadabilityOnlyNote,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			indexRef, packageName, channelName := args[0], args[1], args[2]

			render := action.Render{Refs: []string{indexRef}}
			cfg, err := render.Run(cmd.Context())
			if err != nil {
				logger.Fatal(err)
			}

			m, err := declcfg.ConvertToModel(*cfg)
			if err != nil {
				logger.Fatal(err)
			}
			pkg, ok := m[packageName]
			if !ok {
				logger.Fatalf("package %q not found in index %q", packageName, indexRef)
			}
			ch, ok := pkg.Channels[channelName]
			if !ok {
				logger.Fatalf("channel %q not found in package %q in index %q", channelName, packageName, indexRef)
			}
			g := graphviz.New()
			graph, err := newGraph(*g)
			if err != nil {
				logger.Fatal(err)
			}
			for _, b := range ch.Bundles {
				_ = graph.CreateNode(b.Name)
				if b.Replaces != "" {
					_ = graph.CreateEdge(b.Replaces, b.Name, "")
				}

				for _, skip := range b.Skips {
					_ = graph.CreateEdge(skip, b.Name, "")
				}
			}
			for _, b := range ch.Bundles {
				for _, skip := range b.Skips {
					graph.DeleteEdgesTo(skip)
				}
			}
			if err := g.Render(graph.g, graphviz.SVG, os.Stdout); err != nil {
				logger.Fatal(err)
			}
			return nil
		},
	}
}

type graph struct {
	g         *cgraph.Graph
	nodes     map[string]*cgraph.Node
	edgesFrom map[string][]*cgraph.Edge
	edgesTo   map[string][]*cgraph.Edge
}

func newGraph(g graphviz.Graphviz) (*graph, error) {
	gr, err := g.Graph()
	if err != nil {
		return nil, err
	}
	gr.SetRankDir(cgraph.BTRank)
	return &graph{
		g:         gr,
		nodes:     map[string]*cgraph.Node{},
		edgesFrom: map[string][]*cgraph.Edge{},
		edgesTo:   map[string][]*cgraph.Edge{},
	}, nil
}

func (g *graph) CreateNode(name string) error {
	n, err := g.g.CreateNode(name)
	if err != nil {
		return err
	}
	g.nodes[name] = n
	return nil
}

func (g *graph) CreateEdge(from, to, label string) error {
	fromN, ok := g.nodes[from]
	if !ok {
		if err := g.CreateNode(from); err != nil {
			return err
		}
		fromN = g.nodes[from]
	}

	toN, ok := g.nodes[to]
	if !ok {
		if err := g.CreateNode(to); err != nil {
			return err
		}
		toN = g.nodes[to]
	}
	name := fmt.Sprintf("%s__%s", from, to)
	e, err := g.g.CreateEdge(name, fromN, toN)
	if err != nil {
		return err
	}
	g.edgesFrom[from] = append(g.edgesFrom[from], e)
	g.edgesFrom[to] = append(g.edgesFrom[to], e)

	e.SetLabel(label)
	return nil
}

func (g *graph) DeleteEdgesTo(to string) {
	for _, e := range g.edgesFrom[to] {
		fmt.Fprintf(os.Stderr, "deleting edge %v\n", e.Name())

		g.g.DeleteEdge(e)
	}
}
