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
	"fmt"
	"github.com/go-logr/logr"
	"github.com/noovertime7/kubemonitor/internal/writer"
	"github.com/noovertime7/kubemonitor/pkg/input"
	"github.com/noovertime7/kubemonitor/pkg/types"
	"github.com/noovertime7/kubemonitor/pkg/worker"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubemonitoriov1 "github.com/noovertime7/kubemonitor/api/v1"
)

// MonitorReconciler reconciles a Monitor object
type monitorReconciler struct {
	worker  worker.Worker
	wm      writer.WritersManager
	factory *input.SharedHandlerFactory
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kubemonitor.io.kubemonitor.io,resources=monitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubemonitor.io.kubemonitor.io,resources=monitors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubemonitor.io.kubemonitor.io,resources=monitors/finalizers,verbs=update

func (r *monitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	original := &kubemonitoriov1.Monitor{}
	err := r.Client.Get(ctx, req.NamespacedName, original)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("monitor not found")
			r.worker.Stop(req.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "error getting mysql monitor")
		return ctrl.Result{}, err
	}

	monitor := original.DeepCopy()
	model := monitor.Spec.Model
	logger = logger.WithValues("model", model.Name)

	if err := r.factory.InitConfig(model.Name, model.Config); err != nil {
		logger.Error(err, "init handler config error")
		return ctrl.Result{}, err
	}
	logger.Info("init handler config success")

	r.worker.AddWorkerTask(model.Name)

	err = r.worker.Run(model.Name, monitor.Spec.Period.Duration, func() {
		err := r.factory.Gather(model.Name)
		if err != nil {
			logger.Error(err, "gather error")
			return
		}
		//r.factory.PopBackAll("mysql")
		list, err := r.factory.List(model.Name)
		if err != nil {
			logger.Error(err, "factory list error")
			return
		}
		r.forward(logger, Process(list, monitor.Spec.Labels))
	})
	if err != nil {
		logger.Error(err, "start  monitor error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *monitorReconciler) forward(logger logr.Logger, slist *types.SampleList) {
	if slist == nil {
		logger.Error(fmt.Errorf("data nil"), "")
		return
	}
	arr := slist.PopBackAll()
	r.wm.WriteSamples(arr)
	logger.Info("write samples success", "len", len(arr))
	//printTestMetrics(arr)
}

func NewMonitorReconciler(client client.Client, Scheme *runtime.Scheme, wm writer.WritersManager, worker worker.Worker, factory *input.SharedHandlerFactory) *monitorReconciler {
	return &monitorReconciler{
		worker:  worker,
		Client:  client,
		Scheme:  Scheme,
		wm:      wm,
		factory: factory,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *monitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubemonitoriov1.Monitor{}).
		Complete(r)
}
