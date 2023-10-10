/*
Copyright 2023.

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
	"github.com/noovertime7/kubemonitor/internal/writer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubemonitoriov1 "github.com/noovertime7/kubemonitor/api/v1"
)

// PrometheusPushReconciler reconciles a PrometheusPush object
type prometheusPushReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	wm     writer.WritersManager
}

func NewPrometheusPushReconciler(client client.Client, Scheme *runtime.Scheme, wm writer.WritersManager) *prometheusPushReconciler {
	return &prometheusPushReconciler{
		wm:     wm,
		Client: client,
		Scheme: Scheme,
	}
}

//+kubebuilder:rbac:groups=kubemonitor.io.kubemonitor.io,resources=prometheuspushes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubemonitor.io.kubemonitor.io,resources=prometheuspushes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubemonitor.io.kubemonitor.io,resources=prometheuspushes/finalizers,verbs=update

func (r *prometheusPushReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("controller", PrometheusPush)

	original := &kubemonitoriov1.PrometheusPush{}
	err := r.Client.Get(ctx, req.NamespacedName, original)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("PrometheusPush not found")
			if err = r.wm.DeRegister(req.Name); err != nil {
				logger.Error(err, "DeRegister error,will retry...")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "error getting PrometheusPush")
		return ctrl.Result{}, err
	}

	err = r.wm.Register(req.Name, writer.WriterOption{
		Url:                 original.Spec.Url,
		BasicAuthUser:       original.Spec.BasicAuthUser,
		BasicAuthPass:       original.Spec.BasicAuthPass,
		Headers:             original.Spec.Headers,
		Timeout:             original.Spec.Timeout,
		DialTimeout:         original.Spec.DialTimeout,
		MaxIdleConnsPerHost: original.Spec.MaxIdleConnsPerHost,
	})
	if err != nil {
		logger.Error(err, "register error")
		return ctrl.Result{}, err
	}
	logger.Info("register writer success", "name", req.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *prometheusPushReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubemonitoriov1.PrometheusPush{}).
		Complete(r)
}
