/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package denyhostipc

import (
	"fmt"
	"io"

	"k8s.io/kubernetes/pkg/admission"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/client"
)

func init() {
	admission.RegisterPlugin("DenyExecOnHostIpc", func(client client.Interface, config io.Reader) (admission.Interface, error) {
		return NewDenyExecOnHostIpc(client), nil
	})
}

// denyExecOnHostIpc is an implementation of admission.Interface which says no to a pod/exec on
// a pod using host ipc
type denyExecOnHostIpc struct {
	*admission.Handler
	client client.Interface
}

func (d *denyExecOnHostIpc) Admit(a admission.Attributes) (err error) {
	connectRequest, ok := a.GetObject().(*rest.ConnectRequest)
	if !ok {
		return errors.NewBadRequest("a connect request was received, but could not convert the request object.")
	}
	// Only handle exec requests on pods
	if connectRequest.ResourcePath != "pods/exec" {
		return nil
	}
	pod, err := d.client.Pods(a.GetNamespace()).Get(connectRequest.Name)
	if err != nil {
		return admission.NewForbidden(a, err)
	}
	if isUsingHostIpc(pod) {
		return admission.NewForbidden(a, fmt.Errorf("Cannot exec container using host ipc"))
	}
	return nil
}

// isUsingHostIpc will return true a pod has any container using host ipc
func isUsingHostIpc(pod *api.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext == nil || c.SecurityContext.UseHostIpc == nil {
			continue
		}
		if *c.SecurityContext.UseHostIpc {
			return true
		}
	}
	return false
}

// NewDenyExecOnHostIpc creates a new admission controller that denies an exec operation on a pod using host ipc
func NewDenyExecOnHostIpc(client client.Interface) admission.Interface {
	return &denyExecOnHostIpc{
		Handler: admission.NewHandler(admission.Connect),
		client:  client,
	}
}
