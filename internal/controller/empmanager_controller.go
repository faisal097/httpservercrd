/*
Copyright 2024.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	httpserverv1alpha1 "github.com/faisal097/extending-kubernetes/api/v1alpha1"
)

// EmpManagerReconciler reconciles a EmpManager object
type EmpManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=httpserver.ext-k8s.faisal097.com,resources=empmanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=httpserver.ext-k8s.faisal097.com,resources=empmanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=httpserver.ext-k8s.faisal097.com,resources=empmanagers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EmpManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *EmpManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	instance := &httpserverv1alpha1.EmpManager{}

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Error(err, "unable to get resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.Status.State == "" {
		instance.Status.State = httpserverv1alpha1.PENDING_STATE
		if err := r.Status().Update(ctx, instance); err != nil {
			log.Error(err, "unable to update resource status to PENDING_STATE")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	specBytes := []byte(instance.Spec.Spec)
	objs := parseK8sYaml(ctx, specBytes)

	for _, obj := range objs {
		switch t := obj.(type) {
		case *appsV1.Deployment:
			log.Info(fmt.Sprintf("Deployment found: %s", stringify(ctx, t)))
		case *v1.Service:
			log.Info(fmt.Sprintf("Service found: %s", stringify(ctx, t)))
		case *v1.Namespace:
			log.Info(fmt.Sprintf("Namespace found: %s", stringify(ctx, t)))
		default:
			log.Info(fmt.Sprintf("UnIdentified type: %s", stringify(ctx, t)))
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmpManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&httpserverv1alpha1.EmpManager{}).
		Complete(r)
}

func parseK8sYaml(ctx context.Context, fileR []byte) []runtime.Object {
	log := log.FromContext(ctx)
	acceptedK8sTypes := regexp.MustCompile(`(Namespace|Deployment|Service)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]runtime.Object, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(f), nil, nil)

		if err != nil {
			log.Error(err, "Error while decoding YAML object")
			continue
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			log.Info(fmt.Sprintf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind))
		} else {
			retVal = append(retVal, obj)
		}

	}
	return retVal
}

func stringify(ctx context.Context, object interface{}) string {
	log := log.FromContext(ctx)
	bytes, err := json.Marshal(object)
	if err != nil {
		log.Error(err, "Error while json.Marshal object")
		return ""
	}

	return string(bytes)
}
