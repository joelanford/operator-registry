package configmap

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/containers/storage/pkg/archive"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/operator-framework/operator-registry/pkg/api"
	"github.com/operator-framework/operator-registry/pkg/lib/encoding"
	"github.com/operator-framework/operator-registry/pkg/registry"
)

func NewBundleLoader() *BundleLoader {
	logger := logrus.NewEntry(logrus.New())
	return NewBundleLoaderWithLogger(logger)
}

func NewBundleLoaderWithLogger(logger *logrus.Entry) *BundleLoader {
	return &BundleLoader{
		logger: logger,
	}
}

type BundleLoader struct {
	logger *logrus.Entry
}

// Load accepts a ConfigMap object, iterates through the Data section and
// creates an operator registry Bundle object.
// If the Data section has a PackageManifest resource then it is also
// deserialized and included in the result.
func (l *BundleLoader) Load(cm *corev1.ConfigMap) (*api.Bundle, error) {
	var err error
	if cm == nil {
		err = errors.New("ConfigMap must not be <nil>")
		return nil, err
	}

	logger := l.logger.WithFields(logrus.Fields{
		"configmap": fmt.Sprintf("%s/%s", cm.GetNamespace(), cm.GetName()),
	})

	bundle, skipped, bundleErr := loadBundle(logger, cm)
	if bundleErr != nil {
		err = fmt.Errorf("failed to extract bundle from configmap - %v", bundleErr)
		return nil, err
	}
	l.logger.Debugf("couldn't unpack skipped: %#v", skipped)
	return bundle, nil
}

func loadBundle(entry *logrus.Entry, cm *corev1.ConfigMap) (*api.Bundle, map[string]string, error) {
	bundle := &api.Bundle{Object: []string{}}
	skipped := map[string]string{}

	data := cm.Data
	encoding, ok := data[ConfigMapEncodingAnnotationKey]
	if ok {
		switch encoding {
		case ConfigMapEncodingAnnotationGzip:
			entry.Debug("Decoding gzip-encoded bundle data")
			var err error
			data, err = decodeGzipBinaryData(cm)
			if err != nil {
				return nil, nil, err
			}
		case ConfigMapEncodingAnnotationTarGzip:
			entry.Debug("Decoding tgz-encoded bundle data")
			var err error
			data, err = decodeTarGzBinaryData(cm)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	// Add kube resources to the bundle.
	// Sort keys by name to guarantee consistent ordering of bundle objects.
	for _, name := range slices.Sorted(maps.Keys(data)) {
		content := data[name]
		reader := strings.NewReader(content)
		logger := entry.WithFields(logrus.Fields{
			"key": name,
		})

		resource, decodeErr := registry.DecodeUnstructured(reader)
		if decodeErr != nil {
			logger.Infof("skipping due to decode error - %v", decodeErr)

			// It may not be not a kube resource, let's add it to the skipped
			// list so the caller can act on ot.
			skipped[name] = content
			continue
		}

		if resource.GetKind() == "ClusterServiceVersion" {
			csvBytes, err := resource.MarshalJSON()
			if err != nil {
				return nil, nil, err
			}
			bundle.CsvJson = string(csvBytes)
			bundle.CsvName = resource.GetName()
		}
		bundle.Object = append(bundle.Object, content)
		logger.Infof("added to bundle, Kind=%s", resource.GetKind())
	}

	return bundle, skipped, nil
}

func decodeTarGzBinaryData(cm *corev1.ConfigMap) (map[string]string, error) {
	tmpDir, err := os.MkdirTemp("", "bundle-tar-gzip-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	dataBuf := &bytes.Buffer{}
	bytesBuf := make([]byte, 32*1024)

	data := make(map[string]string)
	gzr, err := gzip.NewReader()
	if err != nil {
		return nil, err
	}

	defer gzr.Close()
	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		dataBuf.Reset()
		if _, err := io.CopyBuffer(dataBuf, tarReader, bytesBuf); err != nil {
			return nil, err
		}
		data[header.Name] = dataBuf.String()
	}
	return data, nil
}

func decodeGzipBinaryData(cm *corev1.ConfigMap) (map[string]string, error) {
	data := map[string]string{}

	for name, content := range cm.BinaryData {
		decoded, err := encoding.GzipBase64Decode(content)
		if err != nil {
			return nil, fmt.Errorf("error decoding gzip-encoded bundle data: %v", err)
		}

		data[name] = string(decoded)
	}

	return data, nil
}
