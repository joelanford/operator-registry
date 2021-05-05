package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/operator-framework/operator-registry/internal/declcfg"
	"github.com/operator-framework/operator-registry/internal/property"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
)

func NewCmd() *cobra.Command {
	output := ""
	cmd := &cobra.Command{
		Use:   "render <index-or-bundle-image1> <index-or-bundle-image2> <index-or-bundle-imageN>",
		Short: "Generate declarative config blobs from the provided index and bundle images",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			refs := args

			var write func(declcfg.DeclarativeConfig, io.Writer) error
			switch output {
			case "yaml":
				write = declcfg.WriteYAML
			case "json":
				write = declcfg.WriteJSON
			default:
				log.Fatalf("invalid --output value %q, expected (json|yaml)", output)
			}
			logger := logrus.New()
			logger.SetOutput(ioutil.Discard)
			nullLogger := logrus.NewEntry(logger)
			logrus.SetOutput(ioutil.Discard)

			cacheDir, err := os.MkdirTemp("", "opm-unpack-")
			if err != nil {
				log.Fatal(err)
			}
			reg, err := containerdregistry.NewRegistry(containerdregistry.WithCacheDir(cacheDir), containerdregistry.WithLog(nullLogger))
			if err != nil {
				log.Fatal(err)
			}
			defer reg.Destroy()

			var out bytes.Buffer
			for _, ref := range refs {
				var cfg *declcfg.DeclarativeConfig
				if stat, serr := os.Stat(ref); serr == nil && stat.IsDir() {
					cfg, err = declcfg.LoadDir(ref)
				} else {
					cfg, err = imageToDeclcfg(cmd.Context(), reg, ref)
				}
				if err != nil {
					log.Fatal(err)
				}
				renderBundleObjects(cfg)
				if err := write(*cfg, &out); err != nil {
					log.Fatal(err)
				}
			}
			if _, err := fmt.Fprint(os.Stdout, out.String()); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "json", "Output format (json|yaml)")
	return cmd
}

func imageToDeclcfg(ctx context.Context, reg *containerdregistry.Registry, imageRef string) (*declcfg.DeclarativeConfig, error) {
	ref := image.SimpleReference(imageRef)
	if err := reg.Pull(ctx, ref); err != nil {
		return nil, err
	}
	labels, err := reg.Labels(ctx, ref)
	if err != nil {
		return nil, err
	}
	tmpDir, err := ioutil.TempDir("", "opm-unpack-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	if err := reg.Unpack(ctx, ref, tmpDir); err != nil {
		return nil, err
	}

	var cfg *declcfg.DeclarativeConfig
	if dbFile, ok := labels[containertools.DbLocationLabel]; ok {
		cfg, err = sqliteToDeclcfg(ctx, filepath.Join(tmpDir, dbFile))
		if err != nil {
			return nil, err
		}
	} else if configsDir, ok := labels["operators.operatorframework.io.index.configs.v1"]; ok {
		cfg, err = declcfg.LoadDir(filepath.Join(tmpDir, configsDir))
		if err != nil {
			return nil, err
		}
	} else if _, ok := labels[bundle.PackageLabel]; ok {
		img, err := registry.NewImageInput(ref, tmpDir)
		if err != nil {
			return nil, err
		}

		cfg, err = bundleToDeclcfg(img.Bundle)
		if err != nil {
			return nil, err
		}
	} else {
		labelKeys := sets.StringKeySet(labels)
		labelVals := []string{}
		for _, k := range labelKeys.List() {
			labelVals = append(labelVals, fmt.Sprintf("  %s=%s", k, labels[k]))
		}
		if len(labelVals) > 0 {
			return nil, fmt.Errorf("unpack %q: image type could not be determined, found labels\n%s", ref, strings.Join(labelVals, "\n"))
		} else {
			return nil, fmt.Errorf("unpack %q: image type could not be determined: image has no labels", ref)
		}
	}
	return cfg, nil
}

func sqliteToDeclcfg(ctx context.Context, dbFile string) (*declcfg.DeclarativeConfig, error) {
	db, err := sqlite.Open(dbFile)
	if err != nil {
		return nil, err
	}

	migrator, err := sqlite.NewSQLLiteMigrator(db)
	if err != nil {
		return nil, err
	}
	if migrator == nil {
		return nil, fmt.Errorf("failed to load migrator")
	}

	if err := migrator.Migrate(ctx); err != nil {
		return nil, err
	}

	q := sqlite.NewSQLLiteQuerierFromDb(db)
	m, err := sqlite.ToModel(ctx, q)
	if err != nil {
		return nil, err
	}

	cfg := declcfg.ConvertFromModel(m)
	return &cfg, nil
}

func bundleToDeclcfg(bundle *registry.Bundle) (*declcfg.DeclarativeConfig, error) {
	bundleProperties, err := registry.PropertiesFromBundle(bundle)
	if err != nil {
		return nil, fmt.Errorf("get properties for bundle %q: %v", bundle.Name, err)
	}
	relatedImages, err := getRelatedImages(bundle)
	if err != nil {
		return nil, fmt.Errorf("get related images for bundle %q: %v", bundle.Name, err)
	}

	dBundle := declcfg.Bundle{
		Schema:        "olm.bundle",
		Name:          bundle.Name,
		Package:       bundle.Package,
		Image:         bundle.BundleImage,
		Properties:    bundleProperties,
		RelatedImages: relatedImages,
	}

	return &declcfg.DeclarativeConfig{Bundles: []declcfg.Bundle{dBundle}}, nil
}

func getRelatedImages(b *registry.Bundle) ([]declcfg.RelatedImage, error) {
	csv, err := b.ClusterServiceVersion()
	if err != nil {
		return nil, err
	}

	var objmap map[string]*json.RawMessage
	if err = json.Unmarshal(csv.Spec, &objmap); err != nil {
		return nil, err
	}

	rawValue, ok := objmap["relatedImages"]
	if !ok || rawValue == nil {
		return nil, err
	}

	var relatedImages []declcfg.RelatedImage
	if err = json.Unmarshal(*rawValue, &relatedImages); err != nil {
		return nil, err
	}
	return relatedImages, nil
}

func renderBundleObjects(cfg *declcfg.DeclarativeConfig) {
	for bi, b := range cfg.Bundles {
		props := b.Properties[:0]
		for _, p := range b.Properties {
			if p.Type != property.TypeBundleObject {
				props = append(props, p)
			}
		}

		for _, obj := range b.Objects {
			props = append(props, property.MustBuildBundleObjectData([]byte(obj)))
		}
		cfg.Bundles[bi].Properties = props
	}
}
