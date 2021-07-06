package list

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/internal/action"
	"github.com/operator-framework/operator-registry/internal/declcfg"
	"github.com/operator-framework/operator-registry/internal/model"
)

func newBundlesCmd() *cobra.Command {
	logger := logrus.New()

	return &cobra.Command{
		Use:   "bundles <indexRef> <packageName>",
		Short: "List package bundles in an index",
		Long: `The "bundles" command lists the bundles from the specified index and package.
Bundles that exist in multiple channels are duplicated in the output (one
for each channel in which the bundle is present).

` + humanReadabilityOnlyNote,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			indexRef, packageName := args[0], args[1]

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

			var bundles []model.Bundle
			for _, ch := range pkg.Channels {
				for _, b := range ch.Bundles {
					bundles = append(bundles, *b)
				}
			}

			sort.Slice(bundles, func(i, j int) bool {
				if bundles[i].Channel.Name != bundles[j].Channel.Name {
					return bundles[i].Channel.Name < bundles[j].Channel.Name
				}
				return bundles[i].Name < bundles[j].Name
			})
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			defer w.Flush()

			fmt.Fprintln(w, "NAME\tCHANNEL\tREPLACES\tSKIPS\tIMAGE")
			for _, b := range bundles {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", b.Name, b.Channel.Name, b.Replaces, strings.Join(b.Skips, ","), b.Image)
			}

			return nil
		},
	}
}
