/*
Copyright 2018 The Volcano Authors.

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

package admission

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"strconv"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

const (
	DefaultQueue = "default"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// mutate job.
func MutateJobs(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.V(3).Infof("mutating jobs")

	job, err := DecodeJob(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return ToAdmissionResponse(err)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	var patchBytes []byte
	switch ar.Request.Operation {
	case v1beta1.Create:
		patchBytes, err = createPatch(job)
		break
	default:
		err = fmt.Errorf("expect operation to be 'CREATE' ")
		return ToAdmissionResponse(err)
	}

	if err != nil {
		reviewResponse.Result = &metav1.Status{Message: err.Error()}
		return &reviewResponse
	}
	glog.V(3).Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	reviewResponse.Patch = patchBytes
	pt := v1beta1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt

	return &reviewResponse
}

func createPatch(job v1alpha1.Job) ([]byte, error) {
	var patch []patchOperation
	pathQueue := patchDefaultQueue(job)
	if pathQueue != nil {
		patch = append(patch, *pathQueue)
	}
	pathSpec := mutateSpec(job.Spec.Tasks, "/spec/tasks")
	if pathSpec != nil {
		patch = append(patch, *pathSpec)
	}
	return json.Marshal(patch)
}

func patchDefaultQueue(job v1alpha1.Job) *patchOperation {
	//Add default queue if not specified.
	if job.Spec.Queue == "" {
		return &patchOperation{Op: "add", Path: "/spec/queue", Value: DefaultQueue}
	}
	return nil
}

func mutateSpec(tasks []v1alpha1.TaskSpec, basePath string) *patchOperation {
	patched := false
	for index := range tasks {
		// add default task name
		taskName := tasks[index].Name
		if len(taskName) == 0 {
			patched = true
			tasks[index].Name = v1alpha1.DefaultTaskSpec + strconv.Itoa(index)
		}
	}
	if !patched {
		return nil
	}
	return &patchOperation{
		Op:    "replace",
		Path:  basePath,
		Value: tasks,
	}
}
