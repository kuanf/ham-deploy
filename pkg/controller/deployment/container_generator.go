// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployment

import (
	toolsv1alpha1 "github.com/hybridapp-io/ham-deploy/pkg/apis/tools/v1alpha1"
	"github.com/hybridapp-io/ham-deploy/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func (r *ReconcileOperator) configContainerByGenericSpec(spec *toolsv1alpha1.GenericContainerSpec, ctn *corev1.Container) *corev1.Container {
	if spec == nil {
		return ctn
	}

	// apply new values from cr if not nil
	if spec.Image != nil {
		ctn.Image = *spec.Image
	}

	if spec.Name != nil {
		ctn.Name = *spec.Name
	}

	if spec.Command != nil {
		ctn.Command = spec.Command
	}

	if spec.Args != nil {
		ctn.Args = spec.Args
	}

	return ctn
}

func (r *ReconcileOperator) generateDeployableContainer(spec *toolsv1alpha1.DeployableOperatorSpec, pod *corev1.Pod) *corev1.Container {
	if spec == nil {
		return nil
	}

	envwatchns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyWATCHNAMESPACE, Value: ""}
	envpn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyPODNAME, Value: pod.Name}
	envon := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyOPERATORNAME, Value: toolsv1alpha1.DefaultDeployableContainerName}

	ctn := &corev1.Container{
		Name:    toolsv1alpha1.DefaultDeployableContainerName,
		Image:   toolsv1alpha1.DefaultDeployableContainerImage,
		Command: toolsv1alpha1.DefaultDeployableContainerCommand,
		Env: []corev1.EnvVar{
			envwatchns,
			envpn,
			envon,
		},
	}

	// install crds for deployable operator if missing
	_ = utils.CheckAndInstallCRDs(r.client, crdRootPath+crdDeployableSubPath)

	ctn = r.configContainerByGenericSpec(&spec.GenericContainerSpec, ctn)

	return ctn
}

func (r *ReconcileOperator) generateAssemblerContainer(spec *toolsv1alpha1.ApplicationAssemblerSpec, pod *corev1.Pod) *corev1.Container {
	envwatchns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyWATCHNAMESPACE, Value: ""}
	envpn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyPODNAME, Value: pod.Name}
	envon := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyOPERATORNAME, Value: toolsv1alpha1.DefaultAssemblerContainerName}

	ctn := &corev1.Container{
		Name:    toolsv1alpha1.DefaultAssemblerContainerName,
		Image:   toolsv1alpha1.DefaultAssemblerContainerImage,
		Command: toolsv1alpha1.DefaultAssemblerContainerCommand,
		Env: []corev1.EnvVar{
			envwatchns,
			envpn,
			envon,
		},
	}

	// install crds for deployable operator if missing
	_ = utils.CheckAndInstallCRDs(r.client, crdRootPath+crdAssemblerSubPath)

	ctn = r.configContainerByGenericSpec(&spec.GenericContainerSpec, ctn)

	return ctn
}

func (r *ReconcileOperator) generateDiscovererContainer(spec *toolsv1alpha1.ResourceDiscovererSpec, pod *corev1.Pod) *corev1.Container {
	envwatchns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyWATCHNAMESPACE, Value: ""}
	envpn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyPODNAME, Value: pod.Name}
	envon := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyOPERATORNAME, Value: toolsv1alpha1.DefaultDiscovererContainerName}

	// apply required env var cluster name and cluster namespace
	envcn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyCLUSTERNAME, Value: spec.ClusterName}
	envcns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyCLUSTERNAMESPACE, Value: spec.ClusterNamespace}

	ctn := &corev1.Container{
		Name:    toolsv1alpha1.DefaultDiscovererContainerName,
		Image:   toolsv1alpha1.DefaultDiscovererContainerImage,
		Command: toolsv1alpha1.DefaultDiscovererContainerCommand,
		Env: []corev1.EnvVar{
			envwatchns,
			envpn,
			envon,
			envcn,
			envcns,
		},
	}

	// install crds for deployable operator if missing
	_ = utils.CheckAndInstallCRDs(r.client, crdRootPath+crdDiscovererSubPath)

	// apply new values from cr if not nil
	ctn = r.configContainerByGenericSpec(&spec.GenericContainerSpec, ctn)

	// hub config setting
	if spec.HubConnectionConfig != nil {
		// kubeconfig
		if spec.HubConnectionConfig.KubeConfig != nil {
			envkc := corev1.EnvVar{
				Name:  toolsv1alpha1.ContainerEnvVarKeyHUBKUBECONFIG,
				Value: *spec.HubConnectionConfig.KubeConfig,
			}
			ctn.Env = append(ctn.Env, envkc)
		}

		if spec.HubConnectionConfig != nil {
			// set up volume
			secvol := corev1.Volume{
				Name: toolsv1alpha1.DefaultPodVolumeNameHubConnection,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: spec.HubConnectionConfig.SecretRef.Name,
					},
				},
			}
			pod.Spec.Volumes = append(pod.Spec.Volumes, secvol)
			// set up mount

			volm := corev1.VolumeMount{
				MountPath: spec.HubConnectionConfig.MountPath,
				Name:      secvol.Name,
			}
			ctn.VolumeMounts = append(ctn.VolumeMounts, volm)
		}
	}

	return ctn
}
