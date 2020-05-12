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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	toolsv1alpha1 "github.com/hybridapp-io/ham-deploy/pkg/apis/tools/v1alpha1"
)

const interval = time.Second * 1

var (
	request = types.NamespacedName{
		Name:      "test",
		Namespace: "default",
	}
)

func TestReconcile(t *testing.T) {
	g := NewWithT(t)

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	g.Expect(add(mgr, rec)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	deploy := &toolsv1alpha1.Deployment{}
	deploy.Name = request.Name
	deploy.Namespace = request.Namespace

	g.Expect(c.Create(context.TODO(), deploy)).To(Succeed())

	pod := &corev1.Pod{}
	podKey := types.NamespacedName{
		Name:      request.Name + "-pod",
		Namespace: request.Namespace,
	}

	time.Sleep(interval)
	g.Expect(c.Get(context.TODO(), podKey, pod)).To(Succeed())

	// delete pod should trigger recreation
	g.Expect(c.Delete(context.TODO(), pod)).To(Succeed())
	time.Sleep(interval)
	g.Expect(c.Get(context.TODO(), podKey, pod)).To(Succeed())

	// delete the deploy first
	g.Expect(c.Get(context.TODO(), request, deploy)).To(Succeed())
	g.Expect(c.Delete(context.TODO(), deploy)).To(Succeed())
	time.Sleep(interval)

	err = c.Get(context.TODO(), request, deploy)
	g.Expect(errors.IsNotFound(err)).To(BeTrue())

	// test api server does not respect delete by ownerreference, so pod won't be automatically deleted
	// can only test delete pod after deploy wont trigger recreation
	g.Expect(c.Delete(context.TODO(), pod)).To(Succeed())
	time.Sleep(interval)

	err = c.Get(context.TODO(), podKey, pod)
	g.Expect(errors.IsNotFound(err)).To(BeTrue())
}