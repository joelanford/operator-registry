package render

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	action2 "github.com/operator-framework/operator-registry/alpha/action"
	declcfg2 "github.com/operator-framework/operator-registry/alpha/declcfg"
)

func NewCmd() *cobra.Command {
	var (
		render action2.Render
		output string
	)
	cmd := &cobra.Command{
		Use:   "render [index-image | bundle-image]...",
		Short: "Generate declarative config blobs from the provided index and bundle images",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			render.Refs = args

			var write func(declcfg2.DeclarativeConfig, io.Writer) error
			switch output {
			case "yaml":
				write = declcfg2.WriteYAML
			case "json":
				write = declcfg2.WriteJSON
			default:
				log.Fatalf("invalid --output value %q, expected (json|yaml)", output)
			}

			// The bundle loading impl is somewhat verbose, even on the happy path,
			// so discard all logrus default logger logs. Any important failures will be
			// returned from render.Run and logged as fatal errors.
			logrus.SetOutput(ioutil.Discard)

			cfg, err := render.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			if err := write(*cfg, os.Stdout); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "json", "Output format (json|yaml)")
	return cmd
}
