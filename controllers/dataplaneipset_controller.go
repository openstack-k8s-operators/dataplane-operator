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

package controllers

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	dataplanev1beta1 "github.com/openstack-k8s-operators/dataplane-operator/api/v1beta1"
	ipam "github.com/openstack-k8s-operators/dataplane-operator/pkg/ipam"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
)

// DataPlaneIPSetReconciler reconciles a DataPlaneIPSet object
type DataPlaneIPSetReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
	Log     logr.Logger
}

//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=dataplaneipsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstacknets,verbs=get;list;watcif k8s_errors.IsNotFound(err) {
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=dataplaneipsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=dataplaneipsets/finalizers,verbs=update
//+kubebuilder:rbac:groups=dataplane.openstack.org,resources=openstacknets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DataPlaneIPSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *DataPlaneIPSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = log.FromContext(ctx)
	result = ctrl.Result{}
	// TODO(user): your logic here
	r.Log.Info("started reconciliation")

	instance := &dataplanev1beta1.DataPlaneIPSet{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return result, nil
		}
		return result, err
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		r.Log,
	)

	if err != nil {
		// helper might be nil, so can't use util.LogErrorForObject since it requires helper as first arg
		r.Log.Error(err, fmt.Sprintf("unable to acquire helper for DataPlaneIPSet %s", instance.Name))
		return result, err
	}

	defer func() {
		if err != nil {
			instance.Status.Conditions.MarkTrue(dataplanev1beta1.DataPlaneIPSetReadyCondition, dataplanev1beta1.DataPlaneIPSetReadyMessage)
		}
		// update the overall status condition if service is ready
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			r.Log.Error(err, "Can't patch Instance")
			_err = err
			return
		}
	}()

	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, err
	}

	if !instance.DeletionTimestamp.IsZero() {
		return r.ReconcileDelete(ctx, instance, helper, req)
	}

	instance.InitCondition()
	// Build AssignIPDetails
	networks := instance.Spec.DataPlaneNetworks

	if instance.Status.IPAddresses == nil {
		instance.Status.IPAddresses = make(map[string]dataplanev1beta1.IPReservation)
	}

	//find the the openstacknet objects
	for _, network := range networks {
		if _, ok := instance.Status.IPAddresses[network.Name]; ok {
			continue
		}
		osnet := &dataplanev1beta1.OpenStackNet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      network.Name,
				Namespace: instance.Namespace,
			},
		}
		err = r.Get(ctx, types.NamespacedName{Name: network.Name, Namespace: instance.Namespace}, osnet)
		if err != nil {
			r.Log.Error(err, "Network Not Found")
			return ctrl.Result{}, err
		}

		aid := ipam.AssignIPDetails{}
		aid.Reservelist = osnet.Status.Reservations
		for _, subnet := range osnet.Status.Subnets {
			if subnet.Name == network.SubnetName {
				_, cidr, err := net.ParseCIDR(subnet.Cidr)
				if err != nil {
					return ctrl.Result{}, err
				}
				aid.IPNet = *cidr
				aid.RangeStart = net.ParseIP(subnet.AllocationRange.AllocationStart)
				aid.RangeEnd = net.ParseIP(subnet.AllocationRange.AllocationEnd)
			}
			ipReservation, updatedReservations, err := ipam.AssignIP(aid, net.ParseIP(network.FixedIP))
			if err != nil {
				return ctrl.Result{}, err
			}
			ipReservation.Gateway = subnet.Gateway
			ipReservation.Routes = subnet.Routes

			r.Log.Info("Got Reservations")
			aid.Reservelist = updatedReservations
			osnet.Status.Reservations = append(osnet.Status.Reservations, ipReservation)
			err = r.Client.Status().Update(ctx, osnet)
			if err != nil {
				r.Log.Info("Failed updating Network Status")
				return ctrl.Result{}, err
			}
			instance.Status.IPAddresses[network.Name] = ipReservation
		}
		if controllerutil.AddFinalizer(osnet, fmt.Sprintf("%s-%s", helper.GetFinalizer(), instance.Name)) {
			err = r.Update(ctx, osnet)
			if err != nil {
				r.Log.Info("Failed Adding finalizer")
				return ctrl.Result{}, err
			}
		}
	}
	instance.Status.Allocated = true
	return ctrl.Result{}, nil
}

// ReconcileDelete reconciles when resource is deleted
func (r *DataPlaneIPSetReconciler) ReconcileDelete(
	ctx context.Context,
	instance *dataplanev1beta1.DataPlaneIPSet,
	helper *helper.Helper,
	req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling IPDataSet delete")
	networks := instance.Spec.DataPlaneNetworks

	for _, network := range networks {
		osnet := &dataplanev1beta1.OpenStackNet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      network.Name,
				Namespace: instance.Namespace,
			},
		}
		err := r.Get(ctx, types.NamespacedName{Name: network.Name, Namespace: instance.Namespace}, osnet)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected.
				// For additional cleanup logic use finalizers. Return and don't requeue.
				continue
			}
			return ctrl.Result{}, err
		}
		// Deallocate IPReservations from network
		reservations := maps.Values(instance.Status.IPAddresses)
		for i, addr := range osnet.Status.Reservations {
			for _, a := range reservations {
				if addr.IP == a.IP {
					r.Log.Info("Deleting IP")
					osnet.Status.Reservations = slices.Delete(osnet.Status.Reservations, i, i+1)
				}
			}
		}

		err = r.Client.Status().Update(ctx, osnet)
		if err != nil {
			r.Log.Info("Failed updating Network Status")
			return ctrl.Result{}, err
		}

		// Remove the finalizers from networks
		if controllerutil.RemoveFinalizer(osnet, fmt.Sprintf("%s-%s", helper.GetFinalizer(), instance.Name)) {
			err := r.Update(ctx, osnet)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	r.Log.Info("Reconciled DataplaneIPSet delete successfully")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataPlaneIPSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dataplanev1beta1.DataPlaneIPSet{}).
		Complete(r)
}
