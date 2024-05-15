package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clusterscansv1 "ansh.spectro/api/v1"
)

// ClusterScanReconciler reconciles a ClusterScan object
type ClusterScanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cluster-scans.ansh.spectro,resources=clusterscans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster-scans.ansh.spectro,resources=clusterscans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster-scans.ansh.spectro,resources=clusterscans/finalizers,verbs=update

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
	// _ = log.FromContext(ctx)
	// TODO(user): your logic here

	logger := log.FromContext(ctx)

	// Fetch the ClusterScan object
	clusterScan := &clusterscansv1.ClusterScan{}
	if err := r.Get(ctx, req.NamespacedName, clusterScan); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Define a new Kubernetes Job based on the ClusterScan
	job := &corev1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterScan.Name + "-job",
			Namespace: clusterScan.Namespace,
			Labels: map[string]string{
				"app": clusterScan.Name,
			},
		},
		Spec: corev1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": clusterScan.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "job-container",
							Image: "busybox",
							Args:  []string{"echo", "Hello from ClusterScan"},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}

	// Set ClusterScan instance as the owner and controller
	if err := ctrl.SetControllerReference(clusterScan, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference for Job")
		return ctrl.Result{}, err
	}

	// Check if the Job already exists
	existingJob := &corev1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, existingJob)
	if err != nil && client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to check if Job exists")
		return ctrl.Result{}, err
	}

	if err != nil {
		// Job doesn't exist, create it
		err = r.Create(ctx, job)
		if err != nil {
			logger.Error(err, "Failed to create Job")
			return ctrl.Result{}, err
		}
		logger.Info("Created Job", "JobName", job.Name)
	}

	logger.Info("Reconciliation successful")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterscansv1.ClusterScan{}).
		Complete(r)
}
