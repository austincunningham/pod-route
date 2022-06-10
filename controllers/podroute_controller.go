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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	routev1 "github.com/openshift/api/route/v1"
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
	//create deployment
	deployment, err := r.createDeployment(cr, r.podRouteDeployment(cr))
	if err != nil {
		return reconcile.Result{}, err
	}
	// If the spec.size in the CR changes, update the deployment number of replicas
	if deployment.Spec.Replicas != &cr.Spec.Replicas {
		controllerutil.CreateOrUpdate(context.TODO(), r.Client, deployment, func() error {
			deployment.Spec.Replicas = &cr.Spec.Replicas
			return nil
		})
	}

	// create service
	err = r.createService(cr, r.podRouteService(cr))
	if err != nil {
		return reconcile.Result{}, err
	}

	// create route
	err = r.createRoute(cr, r.podRouteRoute(cr))
	if err != nil {
		return reconcile.Result{}, err
	}

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
	labels := labels(cr, "backend-PodRoute")
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

// This is the equivalent of creating a service yaml and returning it
// It doesnt create anything on cluster
func (r PodrouteReconciler) podRouteService(cr *quayiov1alpha1.Podroute) *corev1.Service {
	labels := labels(cr, "service-PodRoute")

	podRouteService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podroute-service",
			Namespace: cr.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	controllerutil.SetControllerReference(cr, podRouteService, r.Scheme)
	return podRouteService
}

// This is the equivalent of creating a route yaml file and returning it
// It doesn't create anything on cluster
func (r PodrouteReconciler) podRouteRoute(cr *quayiov1alpha1.Podroute) *routev1.Route {
	labels := labels(cr, "route-PodRoute")

	always200Route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podroute",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "podroute-service",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(8080),
			},
		},
	}

	return always200Route
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

// check for a service if it doesn't exist it creates one on cluster using the service created in podRouteService
func (r PodrouteReconciler) createService(cr *quayiov1alpha1.Podroute, podRouteServcie *corev1.Service) error {
	// check for a deployment in the namespace
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: podRouteServcie.Name, Namespace: cr.Namespace}, found)
	if err != nil {
		log.Log.Info("Creating Service")
		err = r.Client.Create(context.TODO(), podRouteServcie)
		if err != nil {
			log.Log.Error(err, "Failed to create Service")
			return err
		}
	}
	return nil
}

// check for a route if it doesn't exist it creates one on cluster using the route created in podRouteRoute
func (r PodrouteReconciler) createRoute(cr *quayiov1alpha1.Podroute, podRouteRoute *routev1.Route) error {
	// check for a deployment in the namespace
	found := &routev1.Route{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: podRouteRoute.Name, Namespace: cr.Namespace}, found)
	if err != nil {
		log.Log.Info("Creating Route")
		err = r.Client.Create(context.TODO(), podRouteRoute)
		if err != nil {
			log.Log.Error(err, "Failed to create Route")
			return err
		}
	}
	return nil
}
