package configmap

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-registry/internal/util/tar"
	"github.com/operator-framework/operator-registry/pkg/client"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/operator-framework/operator-registry/pkg/lib/encoding"
)

// configmap keys can contain underscores, but configmap names can not
var unallowedKeyChars = regexp.MustCompile("[^-A-Za-z0-9_.]")

const (
	EnvContainerImage                  = "CONTAINER_IMAGE"
	ConfigMapImageAnnotationKey        = "olm.sourceImage"
	ConfigMapEncodingAnnotationKey     = "olm.contentEncoding"
	ConfigMapEncodingAnnotationGzip    = "gzip+base64"
	ConfigMapEncodingAnnotationTarGzip = "tar+gzip"
)

type AnnotationsFile struct {
	Annotations struct {
		Resources      string `json:"operators.operatorframework.io.bundle.manifests.v1"`
		MediaType      string `json:"operators.operatorframework.io.bundle.mediatype.v1"`
		Metadata       string `json:"operators.operatorframework.io.bundle.metadata.v1"`
		Package        string `json:"operators.operatorframework.io.bundle.package.v1"`
		Channels       string `json:"operators.operatorframework.io.bundle.channels.v1"`
		ChannelDefault string `json:"operators.operatorframework.io.bundle.channel.default.v1"`
	} `json:"annotations"`
}

type ConfigMapWriter struct {
	clientset     kubernetes.Interface
	manifestsDir  string
	configMapName string
	namespace     string
	encoding      string
}

func NewConfigMapLoader(configMapName, namespace, manifestsDir, encoding, kubeconfig string) *ConfigMapWriter {
	clientset, err := client.NewKubeClient(kubeconfig, logrus.StandardLogger())
	if err != nil {
		logrus.Fatalf("cluster config failed: %v", err)
	}

	return NewConfigMapLoaderWithClient(configMapName, namespace, manifestsDir, encoding, clientset)
}

func NewConfigMapLoaderWithClient(configMapName, namespace, manifestsDir, encoding string, clientset kubernetes.Interface) *ConfigMapWriter {
	return &ConfigMapWriter{
		clientset:     clientset,
		manifestsDir:  manifestsDir,
		configMapName: configMapName,
		namespace:     namespace,
		encoding:      encoding,
	}
}

func TranslateInvalidChars(input string) string {
	validConfigMapKey := unallowedKeyChars.ReplaceAllString(input, "~")
	return validConfigMapKey
}

func (c *ConfigMapWriter) Populate(maxDataSizeLimit uint64) error {
	configMapPopulate, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(context.TODO(), c.configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	configMapPopulate.Data = map[string]string{}
	configMapPopulate.BinaryData = map[string][]byte{}
	log := logrus.NewEntry(logrus.StandardLogger())

	metadataSubDir := filepath.Join(c.manifestsDir, "metadata")
	annotationsData, err := os.ReadFile(filepath.Join(metadataSubDir, bundle.AnnotationsFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if len(annotationsData) > 0 {
		var annotationsFile AnnotationsFile
		if err := yaml.Unmarshal(annotationsData, &annotationsFile); err != nil {
			return err
		}
		configMapPopulate.SetAnnotations(map[string]string{
			bundle.ManifestsLabel:      annotationsFile.Annotations.Resources,
			bundle.MediatypeLabel:      annotationsFile.Annotations.MediaType,
			bundle.MetadataLabel:       annotationsFile.Annotations.Metadata,
			bundle.PackageLabel:        annotationsFile.Annotations.Package,
			bundle.ChannelsLabel:       annotationsFile.Annotations.Channels,
			bundle.ChannelDefaultLabel: annotationsFile.Annotations.ChannelDefault,
		})
	}
	var totalSize uint64
	switch c.encoding {
	case "", ConfigMapEncodingAnnotationGzip:
		gzipEntries := c.encoding == ConfigMapEncodingAnnotationGzip
		totalSize, err = c.populateEntries(configMapPopulate, gzipEntries, log)
	case ConfigMapEncodingAnnotationTarGzip:
		totalSize, err = c.populateTarGz(configMapPopulate)
	default:
		return fmt.Errorf("unknown encoding %q", c.encoding)
	}
	if err != nil {
		return err
	}

	if c.encoding != "" {
		setEncodingAnnotation(configMapPopulate, c.encoding)
	}

	if totalSize > maxDataSizeLimit {
		log.Errorf("Bundle files totall %d bytes, which exceeds limit of %d", totalSize, maxDataSizeLimit)
		return fmt.Errorf("bundle files total %d bytes, which exceeds limit of %d", totalSize, maxDataSizeLimit)
	}

	if sourceImage := os.Getenv(EnvContainerImage); sourceImage != "" {
		annotations := initAndGetAnnotations(configMapPopulate)
		annotations[ConfigMapImageAnnotationKey] = sourceImage
	}

	if _, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Update(context.TODO(), configMapPopulate, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (c *ConfigMapWriter) populateTarGz(configMapPopulate *corev1.ConfigMap) (uint64, error) {
	var buf bytes.Buffer
	if err := func() error {
		gzw := gzip.NewWriter(&buf)
		defer gzw.Close()
		return tar.WriteFS(gzw, os.DirFS(c.manifestsDir), nil)
	}(); err != nil {
		return 0, err
	}
	configMapPopulate.BinaryData["content"] = buf.Bytes()
	return uint64(buf.Len()), nil
}

func (c *ConfigMapWriter) populateEntries(configMapPopulate *corev1.ConfigMap, gzip bool, log *logrus.Entry) (uint64, error) {
	var totalSize uint64

	manifestsSubDir := "manifests"
	dir := filepath.Join(c.manifestsDir, manifestsSubDir)
	files, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("read dir failed: %v", err)
		return 0, err
	}

	for _, file := range files {
		filePath := filepath.Join(dir, file.Name())
		log := log.WithField("file", filePath)
		log.Info("Reading file")

		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Errorf("read failed: %v", err)
			return 0, err
		}

		if gzip {
			content, err = encoding.GzipBase64Encode(content)
			if err != nil {
				log.Errorf("failed to gzip encode file %v: %v", filePath, err)
				return 0, err
			}
		}

		totalSize += uint64(len(content))

		key := file.Name()
		validConfigMapKey := TranslateInvalidChars(key)
		if validConfigMapKey != key {
			logrus.WithFields(logrus.Fields{
				"file.Name":         key,
				"validConfigMapKey": validConfigMapKey,
			}).Info("translated filename for configmap compatibility")
		}

		if gzip {
			configMapPopulate.BinaryData[validConfigMapKey] = content
		} else {
			configMapPopulate.Data[validConfigMapKey] = string(content)
		}
	}
	return totalSize, nil
}

// LaunchBundleImage will launch a bundle image and also create a configmap for
// storing the data that will be updated to contain the bundle image data. It is
// the responsibility of the caller to delete the job, the pod, and the configmap
// when done. This function is intended to be called from OLM, but is put here
// for locality.
func LaunchBundleImage(kubeclient kubernetes.Interface, bundleImage, initImage, namespace string, gzip bool) (*corev1.ConfigMap, *batchv1.Job, error) {
	// create configmap for bundle image data to write to (will be returned)
	newConfigMap, err := kubeclient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "bundle-image-",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	opmCommand := []string{"/injected/opm", "alpha", "bundle", "extract", "-n", namespace, "-c", newConfigMap.GetName()}
	if gzip {
		opmCommand = append(opmCommand, "--gzip")
	}

	launchJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "deploy-bundle-image-",
		},
		Spec: batchv1.JobSpec{
			//ttlSecondsAfterFinished: 0 // can use in the future to not have to clean up job
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bundle-image",
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:    "bundle-image",
							Image:   bundleImage,
							Command: opmCommand,
							Env: []corev1.EnvVar{
								{
									Name:  EnvContainerImage,
									Value: bundleImage,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "copydir",
									MountPath: "/injected",
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "copy-binary",
							Image:   initImage,
							Command: []string{"/bin/cp", "/bin/opm", "/copy-dest"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "copydir",
									MountPath: "/copy-dest",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "copydir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	launchedJob, err := kubeclient.BatchV1().Jobs(namespace).Create(context.TODO(), &launchJob, metav1.CreateOptions{})
	if err != nil {
		err := kubeclient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), newConfigMap.GetName(), metav1.DeleteOptions{})
		if err != nil {
			// already in an error, so just report it
			logrus.Errorf("failed to remove configmap: %v", err)
		}
		return nil, nil, err
	}

	return newConfigMap, launchedJob, nil
}

func setEncodingAnnotation(cm *corev1.ConfigMap, encoding string) {
	annotations := initAndGetAnnotations(cm)
	annotations[ConfigMapEncodingAnnotationKey] = encoding
}

func hasGzipEncodingAnnotation(cm *corev1.ConfigMap) bool {
	annotations := cm.GetAnnotations()
	encoding, ok := annotations[ConfigMapEncodingAnnotationKey]
	return ok && encoding == ConfigMapEncodingAnnotationGzip
}

func initAndGetAnnotations(cm *corev1.ConfigMap) map[string]string {
	annotations := cm.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
		cm.SetAnnotations(annotations)
	}
	return annotations
}
