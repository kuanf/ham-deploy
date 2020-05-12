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

package utils

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	crdv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

const (
	packageDetailLogLevel = 5
)

func CheckAndInstallCRDs(client client.Client, pathname string) error {
	err := filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		klog.V(packageDetailLogLevel).Info("Working on ", path)
		if info != nil && !info.IsDir() && CheckAndInstallCRD(client, path) != nil {
			return err
		}
		return nil
	})

	return err
}

func CheckAndInstallCRD(client client.Client, pathname string) error {
	var err error

	var crdobj *crdv1beta1.CustomResourceDefinition

	var crddata []byte

	crddata, err = ioutil.ReadFile(pathname)
	if err != nil {
		klog.Error("Loading app crd file", err.Error())
		return err
	}

	err = yaml.Unmarshal(crddata, &crdobj)
	if err != nil {
		klog.Fatal("Unmarshal app crd ", err.Error(), "\n", string(crddata))
		return err
	}

	existcrd := &crdv1beta1.CustomResourceDefinition{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: crdobj.GetName()}, existcrd)

	if errors.IsNotFound(err) {
		klog.Info("Installing SIG Application CRD from File: ", pathname)
		// Install sig app
		err = client.Create(context.TODO(), crdobj)
		if err != nil {
			klog.Error("Creating CRD", err.Error())
			return err
		}
	} else {
		if err != nil {
			klog.Error("Failed to get crd ", crdobj.GetName(), "error:", err)
		}
		klog.V(packageDetailLogLevel).Info("CRD ", crdobj.GetName(), " exists: ", pathname)
	}

	return err
}
