/*
Copyright 2021.

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

package controllers

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	quayiov1alpha1 "github.com/austincunningham/pod-route/api/v1alpha1"
)

// PodrouteReconciler reconciles a Podroute object
type PodrouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=quay.io,resources=podroutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=quay.io,resources=podroutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=quay.io,resources=podroutes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

func (r *PodrouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here
	// Create a Custom Resource object for Podroute, quayio part of the name is due to my earlier mistake
	cr := &quayiov1alpha1.Podroute{}
	// do a kubernetes client get to check if the CR is on the Cluster
	err := r.Client.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		return ctrl.Result{}, err
	}

	deployment, err := r.createDeployment(cr, r.podRouteDeployment(cr))
	if err != nil {
		return reconcile.Result{}, err
	}
	// just logging here to keep Go happy will use later
	log.Log.Info("deployment", deployment)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodrouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&quayiov1alpha1.Podroute{}).
		Complete(r)
}

// don't need this for such a small deployment
// never underestimate a programmers need to complicate simple things
func labels(cr *quayiov1alpha1.Podroute, tier string) map[string]string {
	// Fetches and sets labels

	return map[string]string{
		"app":         "PodRoute",
		"PodRoute_cr": cr.Name,
		"tier":        tier,
	}
}

// This is the equivalent of creating a deployment yaml and returning it
// It doesn't create anything on cluster
func (r *PodrouteReconciler) podRouteDeployment(cr *quayiov1alpha1.Podroute) *appsv1.Deployment {
	// Build a Deployment
	labels := labels(cr, "backend-Podroute")
	size := cr.Spec.Replicas
	podRouteDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "PodRoute",
			Namespace: cr.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           cr.Spec.Image,
						ImagePullPolicy: corev1.PullAlways,
						Name:            "PodRoute-pod",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "PodRoute",
						}},
					}},
				},
			},
		},
	}

	// sets the this controller as owner
	controllerutil.SetControllerReference(cr, podRouteDeployment, r.Scheme)
	return podRouteDeployment
}

// check for a deployment if it doesn't exist it creates one on cluster using the deployment created in deployment
func (r PodrouteReconciler) createDeployment(cr *quayiov1alpha1.Podroute, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	// check for a deployment in the namespace
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: cr.Namespace}, found)
	if err != nil {
		log.Log.Info("Creating deployment")
		err = r.Client.Create(context.TODO(), deployment)
		if err != nil {
			log.Log.Error(err, "Failed to create deployment")
			return found, err
		}
	}
	return found, nil
}
