package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	bolt "go.etcd.io/bbolt"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/pkg/api"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-registry/pkg/registry/cache/internal"
)

var _ Cache = &Bolt{}

type Bolt struct {
	baseDir string

	packageIndex
	db internal.RefCounter[bolt.DB]
}

const (
	boltCacheModeDir  = 0755
	boltCacheModeFile = 0644
)

func (q *Bolt) loadAPIBundle(pkgName, chName, bundleName string) (*api.Bundle, error) {
	var apiBundle api.Bundle

	if err := q.db.With(func(db *bolt.DB) error {
		if err := db.View(func(txn *bolt.Tx) error {
			bucket := txn.Bucket([]byte("bundles"))
			val := bucket.Get([]byte(fmt.Sprintf("%s/%s/%s.proto", pkgName, chName, bundleName)))
			if val == nil {
				return fmt.Errorf("bundle %q not found in channel %q in package %q", bundleName, chName, pkgName)
			}
			return proto.Unmarshal(val, &apiBundle)
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &apiBundle, nil
}

func (q *Bolt) SendBundles(ctx context.Context, stream registry.BundleSender) error {
	return q.db.With(func(db *bolt.DB) error {
		return db.View(func(txn *bolt.Tx) error {
			bucket := txn.Bucket([]byte("bundles"))
			bucket.ForEach(func(_, val []byte) error {
				apiBundle := api.Bundle{}
				if err := proto.Unmarshal(val, &apiBundle); err != nil {
					return err
				}
				if apiBundle.BundlePath != "" {
					// The SQLite-based server
					// configures its querier to
					// omit these fields when
					// bundle path is set.
					apiBundle.CsvJson = ""
					apiBundle.Object = nil
				}
				if err := stream.Send(&apiBundle); err != nil {
					return err
				}
				return nil
			})
			return nil
		})
	})
}

func (q *Bolt) ListBundles(ctx context.Context) ([]*api.Bundle, error) {
	return listBundles(ctx, q)
}

func (q *Bolt) GetBundle(ctx context.Context, pkgName, channelName, csvName string) (*api.Bundle, error) {
	pkg, ok := q.packageIndex[pkgName]
	if !ok {
		return nil, fmt.Errorf("package %q not found", pkgName)
	}
	ch, ok := pkg.Channels[channelName]
	if !ok {
		return nil, fmt.Errorf("package %q, channel %q not found", pkgName, channelName)
	}
	b, ok := ch.Bundles[csvName]
	if !ok {
		return nil, fmt.Errorf("package %q, channel %q, bundle %q not found", pkgName, channelName, csvName)
	}
	apiBundle, err := q.loadAPIBundle(pkg.Name, ch.Name, b.Name)
	if err != nil {
		return nil, fmt.Errorf("load bundle %q: %v", b.Name, err)
	}

	// unset Replaces and Skips (sqlite query does not populate these fields)
	apiBundle.Replaces = ""
	apiBundle.Skips = nil
	return apiBundle, nil
}

func (q *Bolt) GetBundleForChannel(ctx context.Context, pkgName string, channelName string) (*api.Bundle, error) {
	return q.packageIndex.GetBundleForChannel(ctx, q, pkgName, channelName)
}

func (q *Bolt) GetBundleThatReplaces(ctx context.Context, name, pkgName, channelName string) (*api.Bundle, error) {
	return q.packageIndex.GetBundleThatReplaces(ctx, q, name, pkgName, channelName)
}

func (q *Bolt) GetChannelEntriesThatProvide(ctx context.Context, group, version, kind string) ([]*registry.ChannelEntry, error) {
	return q.packageIndex.GetChannelEntriesThatProvide(ctx, q, group, version, kind)
}

func (q *Bolt) GetLatestChannelEntriesThatProvide(ctx context.Context, group, version, kind string) ([]*registry.ChannelEntry, error) {
	return q.packageIndex.GetLatestChannelEntriesThatProvide(ctx, q, group, version, kind)
}

func (q *Bolt) GetBundleThatProvides(ctx context.Context, group, version, kind string) (*api.Bundle, error) {
	return q.packageIndex.GetBundleThatProvides(ctx, q, group, version, kind)
}

func NewBolt(baseDir string) *Bolt {
	return &Bolt{baseDir: baseDir}
}

func (q *Bolt) CheckIntegrity(fbcFsys fs.FS) error {
	existingDigest, err := q.existingDigest()
	if err != nil {
		return fmt.Errorf("read existing cache digest: %v", err)
	}
	computedDigest, err := q.computeDigest(fbcFsys)
	if err != nil {
		return fmt.Errorf("compute digest: %v", err)
	}
	if existingDigest != computedDigest {
		return fmt.Errorf("cache requires rebuild: cache reports digest as %q, but computed digest is %q", existingDigest, computedDigest)
	}
	return nil
}

const (
	boltDigestFile = "bolt.digest"
	boltDBFile     = "bolt.db"
)

func (q *Bolt) existingDigest() (string, error) {
	existingDigestBytes, err := os.ReadFile(filepath.Join(q.baseDir, boltDigestFile))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(existingDigestBytes)), nil
}

func (q *Bolt) computeDigest(fbcFsys fs.FS) (string, error) {
	computedHasher := fnv.New64a()

	// Make sure to include cache format in the digest. This way we will be sure to
	// invalidate caches if/when we change the format.
	computedHasher.Write([]byte("bolt"))
	if err := fsToTar(computedHasher, fbcFsys); err != nil {
		return "", err
	}

	db, err := q.openDBReadOnly()
	if err != nil {
		return "", err
	}
	defer db.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte("packageIndices")).ForEach(func(k, v []byte) error {
			if _, err := computedHasher.Write(k); err != nil {
				return err
			}
			if _, err := computedHasher.Write(v); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		if err := tx.Bucket([]byte("bundles")).ForEach(func(k, v []byte) error {
			if _, err := computedHasher.Write(k); err != nil {
				return err
			}
			if _, err := computedHasher.Write(v); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("compute hash: %v", err)
	}
	return fmt.Sprintf("%x", computedHasher.Sum(nil)), nil
}

func (q *Bolt) Build(fbcFsys fs.FS) error {
	if err := ensureEmptyDir(q.baseDir, boltCacheModeDir); err != nil {
		return fmt.Errorf("ensure clean base directory: %v", err)
	}

	fbc, err := declcfg.LoadFS(fbcFsys)
	if err != nil {
		return err
	}
	fbcModel, err := declcfg.ConvertToModel(*fbc)
	if err != nil {
		return err
	}

	pkgs, err := packagesFromModel(fbcModel)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(q.baseDir, boltDBFile)
	db, err := bolt.Open(dbPath, boltCacheModeFile, nil)
	if err != nil {
		return err
	}

	if err := db.Update(func(txn *bolt.Tx) error {
		pkgIdxBucket, err := txn.CreateBucket([]byte("packageIndices"))
		if err != nil {
			return err
		}
		bundleBucket, err := txn.CreateBucket([]byte("bundles"))
		if err != nil {
			return err
		}
		for _, pkg := range fbcModel {
			pkgIndexBuf := bytes.Buffer{}
			enc := json.NewEncoder(&pkgIndexBuf)
			if err := enc.Encode(pkgs[pkg.Name]); err != nil {
				return err
			}
			pkgIndexKey := fmt.Sprintf("%s.json", pkg.Name)
			if err := pkgIdxBucket.Put([]byte(pkgIndexKey), pkgIndexBuf.Bytes()); err != nil {
				return err
			}
			apiPackage := api.Package{
				Name:               pkg.Name,
				DefaultChannelName: pkg.DefaultChannel.Name,
			}
			for _, ch := range pkg.Channels {
				chHead, err := ch.Head()
				if err != nil {
					return err
				}
				apiPackage.Channels = append(apiPackage.Channels, &api.Channel{
					Name:    ch.Name,
					CsvName: chHead.Name,
				})
				for _, b := range ch.Bundles {
					bKey := fmt.Sprintf("%s/%s/%s.proto", pkg.Name, ch.Name, b.Name)

					apiBundle, err := api.ConvertModelBundleToAPIBundle(*b)
					if err != nil {
						return err
					}

					bundleBytes, err := proto.Marshal(apiBundle)
					if err != nil {
						return err
					}

					if err := bundleBucket.Put([]byte(bKey), bundleBytes); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return err
	}
	digest, err := q.computeDigest(fbcFsys)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(q.baseDir, boltDigestFile), []byte(digest), boltCacheModeFile); err != nil {
		return err
	}
	return nil
}

func (q *Bolt) openDBReadOnly() (*bolt.DB, error) {
	return bolt.Open(filepath.Join(q.baseDir, boltDBFile), boltCacheModeFile, &bolt.Options{
		ReadOnly: true,
	})
}

func (q *Bolt) Load() error {
	rcdb := internal.RefCounter[bolt.DB]{
		Open: q.openDBReadOnly,
		Close: func(db *bolt.DB) error {
			return db.Close()
		},
	}
	pkgs := map[string]cPkg{}

	if err := rcdb.With(func(db *bolt.DB) error {
		if err := db.View(func(txn *bolt.Tx) error {
			bucket := txn.Bucket([]byte("packageIndices"))
			if err := bucket.ForEach(func(_, val []byte) error {
				idxPkg := cPkg{}
				if err := json.NewDecoder(bytes.NewReader(val)).Decode(&idxPkg); err != nil {
					return err
				}
				pkgs[idxPkg.Name] = idxPkg
				return nil
			}); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	q.db = rcdb
	q.packageIndex = pkgs
	return nil
}
