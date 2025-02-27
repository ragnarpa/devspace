package podreplace

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	corev1 "k8s.io/api/core/v1"
	"path"
	"strings"
)

func persistPaths(podName string, replacePod *latest.ReplacePod, copiedPod *corev1.Pod) error {
	name := podName
	if replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.Name != "" {
		name = replacePod.PersistenceOptions.Name
	}

	copiedPod.Spec.Volumes = append(copiedPod.Spec.Volumes, corev1.Volume{
		Name: "devspace-persistence",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: name,
				ReadOnly:  replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.ReadOnly,
			},
		},
	})

	for i, p := range replacePod.PersistPaths {
		if p.Path == "" {
			continue
		}

		subPath := p.VolumePath
		if subPath == "" {
			subPath = fmt.Sprintf("path-%d", i)
		}

		if len(copiedPod.Spec.Containers) > 1 && p.ContainerName == "" {
			names := []string{}
			for _, c := range copiedPod.Spec.Containers {
				names = append(names, c.Name)
			}

			return fmt.Errorf("couldn't persist path %s as multiple containers were found %s, but no containerName was specified", p.Path, strings.Join(names, " "))
		}

		var container *corev1.Container
		for i, con := range copiedPod.Spec.Containers {
			if p.ContainerName == "" || p.ContainerName == con.Name {
				copiedPod.Spec.Containers[i].VolumeMounts = append(copiedPod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "devspace-persistence",
					MountPath: p.Path,
					SubPath:   subPath,
					ReadOnly:  p.ReadOnly,
				})

				container = &con
				break
			}
		}

		if container == nil || p.SkipPopulate || p.ReadOnly || (replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.ReadOnly) {
			continue
		}

		// add an init container that pre-populates the persistent volume for that path
		copiedPod.Spec.InitContainers = append(copiedPod.Spec.InitContainers, corev1.Container{
			Name:    fmt.Sprintf("path-%d-init", i),
			Image:   container.Image,
			Command: []string{"sh"},
			Args:    []string{"-c", fmt.Sprintf("if [ ! -d \"/devspace-persistence/.devspace/\" ]; then cp -a %s/. /devspace-persistence/ && mkdir /devspace-persistence/.devspace ; fi", path.Clean(p.Path))},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "devspace-persistence",
					MountPath: "/devspace-persistence",
					SubPath:   subPath,
				},
			},
		})
	}

	return nil
}
