package init

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/h2non/filetype"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-registry/internal/declcfg"
)

func NewCmd() *cobra.Command {
	var (
		defaultChannel string
		iconFile       string
		description    string
	)
	cmd := &cobra.Command{
		Use:   "init <packageName>",
		Short: "Generate an olm.package declarative config blob",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			packageName := args[0]

			pkg := declcfg.Package{
				Schema:         "olm.package",
				Name:           packageName,
				DefaultChannel: defaultChannel,
				Description:    description,
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
			if err := declcfg.WriteYAML(cfg, os.Stdout); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&defaultChannel, "default-channel", "c", "", "The channel that subscriptions will default to if unspecified")
	cmd.Flags().StringVarP(&iconFile, "icon", "i", "", "Path to package's icon")

	// TODO: support reading description from a file.
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the operator package")
	return cmd
}
