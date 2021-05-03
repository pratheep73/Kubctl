package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterwideNetworkPolicyValidationReconciler validates a ClusterwideNetworkPolicy object
// +kubebuilder:rbac:groups=metal-stack.io,resources=events,verbs=create;patch
type ClusterwideNetworkPolicyValidationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Validates ClusterwideNetworkPolicy object
// +kubebuilder:rbac:groups=metal-stack.io,resources=clusterwidenetworkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metal-stack.io,resources=clusterwidenetworkpolicies/status,verbs=get;update;patch
func (r *ClusterwideNetworkPolicyValidationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	var clusterNP firewallv1.ClusterwideNetworkPolicy
	if err := r.Get(ctx, req.NamespacedName, &clusterNP); err != nil {
		return done, client.IgnoreNotFound(err)
	}

	// if network policy does not belong to the namespace where clusterwide network policies are stored:
	// update status with error message
	if req.Namespace != firewallv1.ClusterwideNetworkPolicyNamespace {
		r.recorder.Event(
			&clusterNP,
			"Warning",
			"Unapplicable",
			fmt.Sprintf("cluster wide network policies must be defined in namespace %s otherwise they won't take effect", firewallv1.ClusterwideNetworkPolicyNamespace),
		)
		return done, nil
	}

	err := clusterNP.Spec.Validate()
	if err != nil {
		r.recorder.Event(
			&clusterNP,
			"Warning",
			"Unapplicable",
			fmt.Sprintf("cluster wide network policy is not valid: %v", err),
		)
		return done, nil
	}

	return done, nil
}

// SetupWithManager configures this controller to watch for ClusterwideNetworkPolicy CRD
func (r *ClusterwideNetworkPolicyValidationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("FirewallController")
	return ctrl.NewControllerManagedBy(mgr).
		For(&firewallv1.ClusterwideNetworkPolicy{}).
		Complete(r)
}