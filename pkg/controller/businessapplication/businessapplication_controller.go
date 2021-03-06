package businessapplication

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/edp-operator/pkg/apis/edp/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new BusinessApplication Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBusinessApplication{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("businessapplication-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BusinessApplication
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.BusinessApplication{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner BusinessApplication
	err = c.Watch(&source.Kind{Type: &batchv1.Job{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &edpv1alpha1.BusinessApplication{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileBusinessApplication{}

// ReconcileBusinessApplication reconciles a BusinessApplication object
type ReconcileBusinessApplication struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a BusinessApplication object and makes changes based on the state read
// and what is in the BusinessApplication.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBusinessApplication) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling BusinessApplication %s/%s\n", request.Namespace, request.Name)

	// Fetch the BusinessApplication instance
	instance := &edpv1alpha1.BusinessApplication{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	updateBusinessApplication(instance, r)

	// Define a new Job object
	job := newJobForCR(instance)

	// Set BusinessApplication instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Job already exists
	found := &batchv1.Job{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating a new Job %s/%s. Command - %s\n", job.Namespace, job.Name, job.Spec.Template.Spec.Containers[0].Command)
		err = r.client.Create(context.TODO(), job)
		if err != nil {

			return reconcile.Result{}, err
		}
		updateStatus(instance, found, r)

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Job already exists - don't requeue
	updateStatus(instance, found, r)
	log.Printf("Skip reconcile: Job %s/%s already exists", found.Namespace, found.Name)
	return reconcile.Result{}, nil
}

func updateBusinessApplication(ba *edpv1alpha1.BusinessApplication, r *ReconcileBusinessApplication) {
	log.Printf("ba.Status.Database.Enabled - %v, ba.Spec.Database - %v", ba.Status.Database.Enabled, ba.Spec.Database)
	if ba.Status.Database.Enabled != ba.Spec.Database {
		log.Printf("Configuration of BA %s has been changed, database enabled has been changed", ba.Name)
		if ba.Spec.Database == true {
			log.Printf("Database was enabled in BA %s", ba.Name)
			log.Printf("Performing enabling database business logic via controller")
			log.Printf("Updating status of BA %s resource to true", ba.Name)
			ba.Status.Database.Enabled = true
			r.client.Update(context.TODO(), ba)
		} else {
			log.Printf("Database was disabled in BA %s", ba.Name)
			log.Printf("Performing disabling database business logic via controller")
			log.Printf("Updating status of BA %s resource to false", ba.Name)
			ba.Status.Database.Enabled = false
			r.client.Update(context.TODO(), ba)
		}
	}
}

func updateStatus(cr *edpv1alpha1.BusinessApplication, job *batchv1.Job, r *ReconcileBusinessApplication) {
	var action, message, status string
	if job.Status.Succeeded > 0 && cr.Status.Message != "Completed" {
		action = "Adding application"
		message = fmt.Sprintf("Business application %s has been successfully added", cr.Name)
		status = "Completed"
	} else {
		action = "Adding application"
		message = fmt.Sprintf("Adding business application %s via Cockpit", cr.Name)
		status = "In progress"
	}
	cr.Status.Action = action
	cr.Status.Message = message
	cr.Status.Status = status
	r.client.Update(context.TODO(), cr)
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newJobForCR(cr *edpv1alpha1.BusinessApplication) *batchv1.Job {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-job",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: "OnFailure",
					Containers: []corev1.Container{
						{
							Name:            "busybox",
							Image:           "busybox",
							Command:         []string{"sleep", "10"},
							ImagePullPolicy: "IfNotPresent",
						},
					},
				},
			},
		},
	}
}
