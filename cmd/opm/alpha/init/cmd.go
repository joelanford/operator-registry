package init

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/h2non/filetype"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/internal/declcfg"
)

func NewCmd() *cobra.Command {
	var (
		defaultChannel  string
		iconFile        string
		descriptionFile string
		output          string
	)
	cmd := &cobra.Command{
		Use:   "init <packageName>",
		Short: "Generate an olm.package declarative config blob",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			packageName := args[0]

			var write func(declcfg.DeclarativeConfig, io.Writer) error
			switch output {
			case "yaml":
				write = declcfg.WriteYAML
			case "json":
				write = declcfg.WriteJSON
			default:
				log.Fatalf("invalid --output value %q, expected (json|yaml)", output)
			}

			pkg := declcfg.Package{
				Schema:         "olm.package",
				Name:           packageName,
				DefaultChannel: defaultChannel,
			}

			if descriptionFile != "" {
				descriptionData, err := ioutil.ReadFile(descriptionFile)
				if err != nil {
					log.Fatalf("read description file %q: %v", iconFile, err)
				}
				pkg.Description = string(descriptionData)
			}

			if iconFile != "" {
				iconData, err := ioutil.ReadFile(iconFile)
				if err != nil {
					log.Fatalf("read icon file %q: %v", iconFile, err)
				}
				iconType, err := filetype.Match(iconData)
				if err != nil {
					log.Fatalf("detect icon mediatype: %v", err)
				}
				if iconType.MIME.Type != "image" {
					log.Fatalf("detected invalid type %q: not an image", iconType.MIME.Value)
				}
				pkg.Icon = &declcfg.Icon{
					Data:      iconData,
					MediaType: iconType.MIME.Value,
				}
			}
			cfg := declcfg.DeclarativeConfig{Packages: []declcfg.Package{pkg}}
			if err := write(cfg, os.Stdout); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&defaultChannel, "default-channel", "c", "", "The channel that subscriptions will default to if unspecified")
	cmd.Flags().StringVarP(&iconFile, "icon", "i", "", "Path to package's icon")
	cmd.Flags().StringVarP(&descriptionFile, "description", "d", "", "Path to the operator's README.md (or other documentation)")
	cmd.Flags().StringVarP(&output, "output", "o", "json", "Output format (json|yaml)")
	return cmd
}
