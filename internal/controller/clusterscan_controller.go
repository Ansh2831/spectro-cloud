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
	"time"

	clusterscansv1 "ansh.spectro/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("controller_clusterscan")

// ClusterScanReconciler reconciles a ClusterScan object
type ClusterScanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=clusterscans.ansh.spectro,resources=clusterscans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=clusterscans.ansh.spectro,resources=clusterscans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=clusterscans.ansh.spectro,resources=clusterscans/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterScan object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *ClusterScanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := logger.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	log.Info("Reconcile called")
	clusterscan := &clusterscansv1.ClusterScan{}

	err := r.Get(ctx, req.NamespacedName, clusterscan)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ClusterScan resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed")
		return ctrl.Result{}, err
	}
	startTime := clusterscan.Spec.Start
	endTime := clusterscan.Spec.End

	currentTime := time.Now().UTC().Hour()

	log.Info(fmt.Sprintf("%d", currentTime))

	if currentTime >= startTime && currentTime <= endTime {

		scaleDeployment(clusterscan, r, ctx, string(clusterscan.Spec.NewSchedule))
	} else {
		scaleDeployment(clusterscan, r, ctx, "*/3 * * * *")
	}
	return ctrl.Result{RequeueAfter: time.Duration(30 * time.Second)}, nil
}

func scaleDeployment(clusterscan *clusterscansv1.ClusterScan, r *ClusterScanReconciler, ctx context.Context, NewSchedule string) error {
	for _, deploy := range clusterscan.Spec.CronJob {
		cron := &batchv1.CronJob{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: deploy.Namespace,
			Name:      deploy.Name,
		}, cron)
		if err != nil {
			return err
		}
		logger.Info("Outside if condition")
		if cron.Spec.Schedule != NewSchedule {
			logger.Info("Inside if condition")
			cron.Spec.Schedule = NewSchedule
			logger.Info("Updatig the cronspec")
			r.Update(ctx, cron)
			err = r.Status().Update(ctx, clusterscan)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterscansv1.ClusterScan{}).
		Complete(r)
}
