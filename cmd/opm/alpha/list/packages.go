package list

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/internal/action"
	"github.com/operator-framework/operator-registry/internal/declcfg"
	"github.com/operator-framework/operator-registry/internal/model"
)

func newPackagesCmd() *cobra.Command {
	logger := logrus.New()

	return &cobra.Command{
		Use:   "packages <indexRef>",
		Short: "List packages in an index",
		Long: `The "channels" command lists the channels from the specified index.

` + humanReadabilityOnlyNote,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			indexRef := args[0]

			render := action.Render{Refs: []string{indexRef}}
			cfg, err := render.Run(cmd.Context())
			if err != nil {
				logger.Fatal(err)
			}

			m, err := declcfg.ConvertToModel(*cfg)
			if err != nil {
				logger.Fatal(err)
			}

			pkgs := []model.Package{}
			for _, pkg := range m {
				pkgs = append(pkgs, *pkg)
			}
			sort.Slice(pkgs, func(i, j int) bool {
				return pkgs[i].Name < pkgs[j].Name
			})

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			defer w.Flush()
			fmt.Fprintln(w, "NAME\tDISPLAY NAME\tDEFAULT CHANNEL")
			for _, pkg := range pkgs {
				fmt.Fprintf(w, "%s\t%s\t%s\n", pkg.Name, getDisplayName(pkg), pkg.DefaultChannel.Name)
			}
			return nil
		},
	}
}

func getDisplayName(pkg model.Package) string {
	if pkg.DefaultChannel == nil {
		return ""
	}
	head, err := pkg.DefaultChannel.Head()
	if err != nil || head == nil || head.CsvJSON == "" {
		return ""
	}

	csv := v1alpha1.ClusterServiceVersion{}
	if err := json.Unmarshal([]byte(head.CsvJSON), &csv); err != nil {
		return ""
	}
	return csv.Spec.DisplayName
}
