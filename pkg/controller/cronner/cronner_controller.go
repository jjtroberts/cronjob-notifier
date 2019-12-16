package cronner

import (
	"context"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	notifyv1alpha1 "github.com/jjtroberts/cronner-operatorr/pkg/apis/notify/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_cronner")

// Add creates a new Cronner Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCronner{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cronner-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Cronner
	err = c.Watch(&source.Kind{Type: &notifyv1alpha1.Cronner{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCronner implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCronner{}

// ReconcileCronner reconciles a Cronner object
type ReconcileCronner struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cronner object and makes changes based on the state read
// and what is in the Cronner.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCronner) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cronner")

	// Fetch the Cronner instance
	instance := &notifyv1alpha1.Cronner{}
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

	// Get most recent failed job pod
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(request.Namespace),
		// client.MatchingFieldsSelector{
		// 	Selector: fields.OneTermEqualSelector("status.phase", "Failed"),
		// },
	}
	err = r.client.List(context.TODO(), podList, listOpts...)
	//reqLogger.Info("Pods:", "PodList", podList)

	pod := getNewestFailedPod(podList.Items, reqLogger, instance.Spec.CronjobName)
	reqLogger.Info("Failed Pod:", "Name", pod)

	// If pod not null
	// --If currentPodID != last_failed_cronjob_id
	// ----retrieve the failed cronjob logs
	// ----write logs to temp file
	// ----Send an e-mail using AWS SES
	// ----Patch cronner

	// Output CR field
	reqLogger.Info("Skip reconcile: testing", "Namespace", request.Namespace, "CronJob.Name", instance.Spec.CronjobName)
	return reconcile.Result{}, nil
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func getNewestFailedPod(pods []corev1.Pod, reqLogger logr.Logger, CronjobName string) corev1.Pod {
	failedPod := corev1.Pod{}
	startTime := "2006-01-02T15:04:05.000Z"
	for _, pod := range pods {
		result, err := regexp.MatchString(CronjobName, pod.Name)

		if err != nil {
			//Check your error here
		}

		if pod.Status.Phase == "Failed" && result {
			reqLogger.Info("Failed:", "Pod.Name", pod.Name)
			myTime := failedPod.GetCreationTimestamp().Local().String()
			if !CheckDateBoundariesStr(startTime, myTime) {
				startTime = myTime
				failedPod = pod
			}
		}
	}
	return failedPod
}

//CheckDateBoundariesStr checks is startdate >= enddate
func CheckDateBoundariesStr(startdate, enddate string) bool {

	layout := "2006-01-02T15:04:05.000Z"

	tstart, err := time.Parse(layout, startdate)
	if err != nil {
		return false //, fmt.Errorf("cannot parse startdate: %v", err)
	}
	tend, err := time.Parse(layout, enddate)
	if err != nil {
		return false //, fmt.Errorf("cannot parse enddate: %v", err)
	}

	if tstart.Before(tend) {
		return false //, fmt.Errorf("startdate < enddate - please set proper data boundaries")
	}
	return true //, err
}
