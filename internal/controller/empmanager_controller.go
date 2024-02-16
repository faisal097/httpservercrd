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
	"fmt"
	"reflect"

	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
		if errors.IsNotFound(err) {
			//CRD is already deleted
			log.Error(err, "CRD instance does not exists")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
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
	var desiredDeploy *appsV1.Deployment
	var desiredService *v1.Service
	for _, obj := range objs {
		switch t := obj.(type) {
		case *appsV1.Deployment:
			desiredDeploy = t
			log.Info(fmt.Sprintf("Deployment found: %s", stringify(ctx, t)))
		case *v1.Service:
			desiredService = t
			log.Info(fmt.Sprintf("Service found: %s", stringify(ctx, t)))
		case *v1.Namespace:
			log.Info(fmt.Sprintf("Namespace found: %s", stringify(ctx, t)))
		default:
			log.Info(fmt.Sprintf("UnIdentified type: %s", stringify(ctx, t)))
		}
	}

	desiredDeploy.Spec.Replicas = &instance.Spec.Replicas
	desiredDeploy.Spec.Template.Spec.Containers[0].Image = instance.Spec.Image

	err := controllerutil.SetControllerReference(instance, desiredDeploy, r.Scheme)
	if err != nil {
		log.Error(err, "Error in setting SetControllerReference for deploy")
		return reconcile.Result{}, err
	}

	if result, err := r.reconcileDeployment(ctx, req, desiredDeploy); err != nil {
		return result, err
	}

	log.Info("Deployemnt is upto date")

	err = controllerutil.SetControllerReference(instance, desiredService, r.Scheme)
	if err != nil {
		log.Error(err, "Error in setting SetControllerReference for service")
		return reconcile.Result{}, err
	}

	if result, err := r.reconcileService(ctx, req, desiredService); err != nil {
		return result, err
	}
	log.Info("Service is upto date")

	instance.Status.State = httpserverv1alpha1.CREATED_STATE
	if err := r.Status().Update(ctx, instance); err != nil {
		log.Error(err, "unable to update resource status to CREATED_STATE")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmpManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&httpserverv1alpha1.EmpManager{}).
		Complete(r)
}

func (r *EmpManagerReconciler) reconcileDeployment(ctx context.Context, req ctrl.Request, desiredDeploy *appsV1.Deployment) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	foundDeploy := &appsV1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      desiredDeploy.Name,
		Namespace: desiredDeploy.Namespace,
	}, foundDeploy)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment %s/%s\n", desiredDeploy.Namespace, desiredDeploy.Name)
		err = r.Create(ctx, desiredDeploy)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error in creating deployment %s/%s", desiredDeploy.Namespace, desiredDeploy.Name))
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, fmt.Sprintf("Error in get deployment %s/%s", desiredDeploy.Namespace, desiredDeploy.Name))
		return reconcile.Result{}, err
	}

	//Check if anything changed in deployment
	if !reflect.DeepEqual(desiredDeploy.Spec, foundDeploy.Spec) {
		foundDeploy.Spec = desiredDeploy.Spec
		log.Info("Updating Deployment %s/%s\n", desiredDeploy.Namespace, desiredDeploy.Name)
		err = r.Update(ctx, foundDeploy)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *EmpManagerReconciler) reconcileService(ctx context.Context, req ctrl.Request, desiredService *v1.Service) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	foundDeploy := &v1.Service{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      desiredService.Name,
		Namespace: desiredService.Namespace,
	}, foundDeploy)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service %s/%s\n", desiredService.Namespace, desiredService.Name)
		err = r.Create(ctx, desiredService)
		if err != nil {
			log.Error(err, fmt.Sprintf("Error in creating Service %s/%s", desiredService.Namespace, desiredService.Name))
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, fmt.Sprintf("Error in get Service %s/%s", desiredService.Namespace, desiredService.Name))
		return reconcile.Result{}, err
	}

	//Check if anything changed in Service
	if !reflect.DeepEqual(desiredService.Spec, desiredService.Spec) {
		foundDeploy.Spec = desiredService.Spec
		log.Info("Updating Service %s/%s\n", desiredService.Namespace, desiredService.Name)
		err = r.Update(ctx, foundDeploy)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
