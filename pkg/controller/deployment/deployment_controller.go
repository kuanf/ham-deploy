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
	"context"

	toolsv1alpha1 "github.com/hybridapp-io/ham-deploy/pkg/apis/tools/v1alpha1"
	"github.com/hybridapp-io/ham-deploy/pkg/utils"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	crdRootPath          = "/usr/local/etc/hybridapp/crds/"
	crdDeployableSubPath = "core/deployable"
	crdAssemblerSubPath  = "tools/assembler"
	crdDiscovererSubPath = "tools/discoverer"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Deployment Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Deployment
	err = c.Watch(&source.Kind{Type: &toolsv1alpha1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Deployment
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &toolsv1alpha1.Deployment{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDeployment implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDeployment{}

// ReconcileDeployment reconciles a Deployment object
type ReconcileDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Deployment object and makes changes based on the state read
// and what is in the Deployment.Spec
func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling Deployment: ", request)

	// Fetch the Deployment instance
	instance := &toolsv1alpha1.Deployment{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	pod := r.newPodForCR(instance)

	// Set Deployment instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		klog.Info("Creating a new Pod Namespace:", pod.Namespace, " Name:", pod.Name)
		err = r.client.Create(context.TODO(), pod)

		if err != nil {
			klog.Error("Failed to create new pod, error:", err)
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - try to update
	uptodate := true
	if !isEqualPod(found, pod) {
		uptodate = false
	}

	if !uptodate {
		err = r.client.Delete(context.TODO(), found)
		if err != nil {
			klog.Error("Failed to delete existin pod with error:", err)
		}

		return reconcile.Result{}, err
	}

	// update deployment status
	found.Status.DeepCopyInto(&instance.Status.PodStatus)
	err = r.client.Status().Update(context.TODO(), found)

	return reconcile.Result{}, err
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func (r *ReconcileDeployment) newPodForCR(cr *toolsv1alpha1.Deployment) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
		},
	}

	pod.Spec.ServiceAccountName = toolsv1alpha1.DefaultPodServiceAccountName

	// inherit operator settings if possible
	opns, err := k8sutil.GetOperatorNamespace()
	if err == nil {
		oppod, err := k8sutil.GetPod(context.TODO(), r.client, opns)
		if err != nil {
			oppod.Spec.DeepCopyInto(&pod.Spec)
		}
	}

	containers := []corev1.Container{}

	// add deployable container unless spec.CoreSpec.DeployableOperatorSpec.Enabled = false
	exists := cr.Spec.CoreSpec == nil || cr.Spec.CoreSpec.DeployableOperatorSpec == nil || cr.Spec.CoreSpec.DeployableOperatorSpec.Enabled == nil
	if exists || *(cr.Spec.CoreSpec.DeployableOperatorSpec.Enabled) {
		envwatchns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyWATCHNAMESPACE, Value: ""}
		envpn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyPODNAME, Value: pod.Name}
		envon := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyOPERATORNAME, Value: toolsv1alpha1.DefaultDeployableContainerName}

		ctn := corev1.Container{
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

		// apply new values from cr if not nil
		containers = append(containers, ctn)
	}

	// add assembler container unless spec.ToolsSpec.ApplicationAssemblerSpec.Enabled = false
	exists = cr.Spec.ToolsSpec == nil || cr.Spec.ToolsSpec.ApplicationAssemblerSpec == nil || cr.Spec.ToolsSpec.ApplicationAssemblerSpec.Enabled == nil
	if exists || *(cr.Spec.ToolsSpec.ApplicationAssemblerSpec.Enabled) {
		envwatchns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyWATCHNAMESPACE, Value: ""}
		envpn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyPODNAME, Value: pod.Name}
		envon := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyOPERATORNAME, Value: toolsv1alpha1.DefaultAssemblerContainerName}

		ctn := corev1.Container{
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

		// apply new values from cr if not nil
		containers = append(containers, ctn)
	}

	// add discoverer container only if spec.ToolsSpec.ResourceDiscovererSpec.Enabled =
	exists = cr.Spec.ToolsSpec != nil && cr.Spec.ToolsSpec.ResourceDiscovererSpec != nil && cr.Spec.ToolsSpec.ResourceDiscovererSpec.Enabled != nil
	if exists && *(cr.Spec.ToolsSpec.ResourceDiscovererSpec.Enabled) {
		envwatchns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyWATCHNAMESPACE, Value: ""}
		envpn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyPODNAME, Value: pod.Name}
		envon := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyOPERATORNAME, Value: toolsv1alpha1.DefaultDiscovererContainerName}

		envcn := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyCLUSTERNAME, Value: cr.Spec.ToolsSpec.ResourceDiscovererSpec.ClusterName}
		envcns := corev1.EnvVar{Name: toolsv1alpha1.ContainerEnvVarKeyCLUSTERNAMESPACE, Value: cr.Spec.ToolsSpec.ResourceDiscovererSpec.ClusterNamespace}

		ctn := corev1.Container{
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
		containers = append(containers, ctn)
	}

	pod.Spec.Containers = containers

	return pod
}

func isEqualPod(oldpod, newpod *corev1.Pod) bool {
	oldimages := make(map[string]string)

	for _, ctn := range oldpod.Spec.Containers {
		oldimages[ctn.Name] = ctn.Image
	}

	for _, ctn := range newpod.Spec.Containers {
		oimg, ok := oldimages[ctn.Name]
		if !ok {
			return false
		}

		if oimg != ctn.Image {
			return false
		}

		delete(oldimages, ctn.Name)
	}

	return len(oldimages) == 0
}
