package list

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List contents of an index",
		Long: `The list subcommands print the contents of an index.

` + humanReadabilityOnlyNote,
	}
	list.AddCommand(newPackagesCmd(), newChannelsCmd(), newBundlesCmd())
	return list
}
