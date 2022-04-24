/*
Copyright 2022.

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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1 "github.com/buddiewei/dkron-operator/api/v1"
	dkron "github.com/buddiewei/dkron-operator/pkg/dkron"
)

var (
	DKRON_SINGLE_CONFIGMAP_NAME string             = "dkron-single-config"
	DKRON_SERVER_CONFIGMAP_NAME string             = "dkron-server-config"
	DKRON_INIT_STS_NAME         string             = "dkron-init"
	DKRON_SERVER_STS_NAME       string             = "dkron-server"
	DKRON_SERVER_SERVICE_NAME   string             = "dkron-server"
	PATH_TYPE_PREFIX            networkv1.PathType = networkv1.PathTypePrefix
)

// DkronOperatorReconciler reconciles a DkronOperator object
type DkronOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.github.com,resources=dkronoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.github.com,resources=dkronoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.github.com,resources=dkronoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DkronOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DkronOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling DkronOperator")

	dkronOperator := &operatorv1.DkronOperator{}
	err := r.Client.Get(ctx, req.NamespacedName, dkronOperator)
	if err != nil {
		logger.Error(err, "unable to fetch DkronOperator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("DkronOperator ImageInfo: ", "ImageName", dkronOperator.Spec.ImageName, "ImageTag", dkronOperator.Spec.ImageTag)

	singleCM, singleCMErr := dkron.BuildConfigMap(ctx, dkronOperator, DKRON_SINGLE_CONFIGMAP_NAME, 1)
	if singleCMErr != nil {
		logger.Error(singleCMErr, "unable to build single configmap")
		return ctrl.Result{}, singleCMErr
	}
	_, couErr := controllerutil.CreateOrUpdate(ctx, r.Client, singleCM, func() error {
		controllerutil.SetControllerReference(dkronOperator, singleCM, r.Scheme)
		return nil
	})
	if couErr != nil {
		logger.Error(couErr, "unable to create or update configmap")
		return ctrl.Result{}, couErr
	}

	exist, err := r.CheckStatefulSetExist(ctx, dkronOperator.Spec.Namespace, DKRON_SERVER_STS_NAME)
	if dkronOperator.Spec.Replicas != 1 && errors.IsNotFound(err) && !exist {
		initSts, initErr := dkron.BuildStatefulSet(ctx, dkronOperator, DKRON_INIT_STS_NAME, 1)
		if initErr != nil {
			logger.Error(initErr, "unable to build StatefulSet")
			return ctrl.Result{}, initErr
		}
		_, couErr := controllerutil.CreateOrUpdate(ctx, r.Client, initSts, func() error {
			controllerutil.SetControllerReference(dkronOperator, initSts, r.Scheme)
			return nil
		})
		if couErr != nil {
			logger.Error(couErr, "unable to create init StatefulSet")
			return ctrl.Result{}, couErr
		}
	}

	serverCM, serverCMErr := dkron.BuildConfigMap(ctx, dkronOperator, DKRON_SERVER_CONFIGMAP_NAME, dkronOperator.Spec.DkronConfig.BootstrapExpect)
	if serverCMErr != nil {
		logger.Error(serverCMErr, "unable to build ConfigMap")
		return ctrl.Result{}, serverCMErr
	}
	_, couErr = controllerutil.CreateOrUpdate(ctx, r.Client, serverCM, func() error {
		controllerutil.SetControllerReference(dkronOperator, serverCM, r.Scheme)
		return nil
	})
	if couErr != nil {
		logger.Error(couErr, "unable to create or update ConfigMap")
		return ctrl.Result{}, couErr
	}

	serverSts, serverErr := dkron.BuildStatefulSet(ctx, dkronOperator, DKRON_SERVER_STS_NAME, dkronOperator.Spec.Replicas)
	if serverErr != nil {
		logger.Error(serverErr, "unable to build StatefulSet")
		return ctrl.Result{}, serverErr
	}
	_, couErr = controllerutil.CreateOrUpdate(ctx, r.Client, serverSts, func() error {
		controllerutil.SetControllerReference(dkronOperator, serverSts, r.Scheme)
		return nil
	})
	if couErr != nil {
		logger.Error(couErr, "unable to create or update StatefulSet")
		return ctrl.Result{}, couErr
	}

	service, serviceErr := dkron.BuildService(ctx, dkronOperator)
	if serviceErr != nil {
		logger.Error(serviceErr, "unable to build Service")
		return ctrl.Result{}, serviceErr
	}
	_, couErr = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		controllerutil.SetControllerReference(dkronOperator, service, r.Scheme)
		return nil
	})
	if couErr != nil {
		logger.Error(couErr, "unable to create or update Service")
		return ctrl.Result{}, couErr
	}
	dkronOperator.Status.ServiceUri = fmt.Sprintf("%s.%s", service.Name, service.Namespace)
	upErr := r.Client.Status().Update(ctx, dkronOperator)
	if upErr != nil {
		logger.Error(upErr, "unable to update status")
		return ctrl.Result{}, upErr
	}

	if dkronOperator.Spec.EnableIngress {
		ingress, ingressErr := dkron.BuildIngress(ctx, dkronOperator)
		if ingressErr != nil {
			logger.Error(ingressErr, "unable to build Ingress")
			return ctrl.Result{}, ingressErr
		}
		_, couErr = controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			controllerutil.SetControllerReference(dkronOperator, ingress, r.Scheme)
			return nil
		})
		if couErr != nil {
			logger.Error(couErr, "unable to create or update Ingress")
			return ctrl.Result{}, couErr
		}
		dkronOperator.Status.IngressUri = ingress.Spec.Rules[0].Host
		upErr := r.Client.Status().Update(ctx, dkronOperator)
		if upErr != nil {
			logger.Error(upErr, "unable to update status")
			return ctrl.Result{}, upErr
		}
	}

	initExist, err := r.CheckStatefulSetExist(ctx, dkronOperator.Spec.Namespace, DKRON_INIT_STS_NAME)
	logger.Info("initExist Info", "initExist", initExist)
	if initExist && err == nil {
		logger.Info("init StatefulSet exist")
		err = r.Client.Get(ctx, types.NamespacedName{Name: DKRON_SERVER_STS_NAME, Namespace: dkronOperator.Spec.Namespace}, serverSts)
		if err != nil {
			logger.Error(err, "unable to get server StatefulSet")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		if isStatefulSetReady(*serverSts, int(dkronOperator.Spec.Replicas)) {
			logger.Info("server StatefulSet is ready, will delete init StatefulSet")
			delErr := r.Client.Delete(ctx, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DKRON_INIT_STS_NAME,
					Namespace: serverSts.Namespace,
				},
			})
			if delErr != nil {
				logger.Error(delErr, "unable to delete StatefulSet")
				return ctrl.Result{}, delErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	} else if !errors.IsNotFound(err) {
		logger.Error(err, "unable to check init StatefulSet exist")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	logger.Info("everything is ok")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DkronOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.DkronOperator{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *DkronOperatorReconciler) CheckStatefulSetExist(ctx context.Context, namespace, stsName string) (bool, error) {
	sts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: namespace}, sts)
	return err == nil && sts.ObjectMeta.DeletionTimestamp == nil, err
}

func isStatefulSetReady(sts appsv1.StatefulSet, expectedReplicas int) bool {
	allUpdated := int32(expectedReplicas) == sts.Status.UpdatedReplicas
	allReady := int32(expectedReplicas) == sts.Status.ReadyReplicas
	atExpectedGeneration := sts.Generation == sts.Status.ObservedGeneration
	return allUpdated && allReady && atExpectedGeneration
}
