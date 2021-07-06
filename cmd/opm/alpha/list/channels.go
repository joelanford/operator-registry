package list

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/internal/action"
	"github.com/operator-framework/operator-registry/internal/declcfg"
	"github.com/operator-framework/operator-registry/internal/model"
)

func newChannelsCmd() *cobra.Command {
	logger := logrus.New()

	return &cobra.Command{
		Use:   "channels <directory> <packageName>",
		Short: "List package channels in an index",
		Long: `The "channels" command lists the channels from the specified index and package.

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

			channels := []model.Channel{}
			for _, ch := range pkg.Channels {
				channels = append(channels, *ch)
			}

			sort.Slice(channels, func(i, j int) bool {
				return channels[i].Name < channels[j].Name
			})
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			defer w.Flush()

			fmt.Fprintln(w, "NAME\tHEAD")
			for _, c := range channels {
				headStr := ""
				head, err := c.Head()
				if err != nil {
					headStr = fmt.Sprintf("ERROR: %s", err)
				} else {
					headStr = head.Name
				}
				fmt.Fprintf(w, "%s\t%s\n", c.Name, headStr)
			}

			return nil
		},
	}
}
