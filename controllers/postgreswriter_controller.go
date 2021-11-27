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
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "github.com/RinkiyaKeDad/second-operator/api/v1"
	"github.com/RinkiyaKeDad/second-operator/pkg/psql"
	"github.com/go-logr/logr"
	"github.com/juju/errors"
)

// PostgresWriterReconciler reconciles a PostgresWriter object
type PostgresWriterReconciler struct {
	// used to talk to the cluster and do CRUD operations on it
	client.Client
	// use to register this type with the corresponding GVK
	// via the manager
	Scheme           *runtime.Scheme
	PostgresDBClient *psql.PostgresDBClient
}

var (
	finalizers []string = []string{"finalizers.postgreswriters.batch.arshsharma.com/cleanup-row"}
)

//+kubebuilder:rbac:groups=batch.arshsharma.com,resources=postgreswriters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.arshsharma.com,resources=postgreswriters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.arshsharma.com,resources=postgreswriters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PostgresWriter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile

// Blog on finalizers: https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/

// TLDR: whenever we delete something it gets updated with a deletion timestamp but the
// deletion will not actually complete until we edit the object and remove any finalizers
// it might have. Till then the object is put it into a read-only state.

// This method gets triggered whenever anything like create/update/delete happens with the
// PostgresWriter custom resource
func (r *PostgresWriterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// parsing the incoming postgres-writer resource
	postgresWriterObject := &batchv1.PostgresWriter{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, postgresWriterObject)
	if err != nil {
		// if the resource is not found, then just return (might look useless as this usually happens in case of Delete events)
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error occurred while fetching the PostgresWriter resource")
		return ctrl.Result{}, err
	}

	// if the event is not related to delete, just check if the finalizers are rightfully set on the resource
	if postgresWriterObject.GetDeletionTimestamp().IsZero() && !reflect.DeepEqual(finalizers, postgresWriterObject.GetFinalizers()) {
		// set the finalizers of the postgresWriterObject to the rightful ones
		postgresWriterObject.SetFinalizers(finalizers)
		if err := r.Update(ctx, postgresWriterObject); err != nil {
			logger.Error(err, "error occurred while setting the finalizers of the PostgresWriter resource")
			return ctrl.Result{}, err
		}
	}

	// if the metadata.deletionTimestamp is found to be non-zero, this means that the resource is intended and just about to be deleted
	// hence, it's time to clean up the finalizers
	if !postgresWriterObject.GetDeletionTimestamp().IsZero() {
		logger.Info("Deletion detected! Proceeding to cleanup the finalizers...")
		if err := r.cleanupRowFinalizerCallback(ctx, logger, postgresWriterObject); err != nil {
			logger.Error(err, "error occurred while dealing with the cleanup-row finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// parsing the table, name, age, country fields from the spec of the incoming postgres-writer resource
	table, name, age, country := postgresWriterObject.Spec.Table, postgresWriterObject.Spec.Name, postgresWriterObject.Spec.Age, postgresWriterObject.Spec.Country

	// forming a unique id corresponding to the incoming resource
	id := postgresWriterObject.Namespace + "/" + postgresWriterObject.Name

	logger.Info(fmt.Sprintf("Writing name: %s age: %d country: %s into table: %s with id as %s ", name, age, country, table, id))

	// performing the `INSERT` to the DB with the provided name, age, country on the provided table
	if err = r.PostgresDBClient.Insert(id, table, name, age, country); err != nil {
		logger.Error(err, "error occurred while inserting the row in the Postgres DB")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.

// SetupWithManager ensures to call the Reconcile method whenever something happens with
// PostgresWriter resource in our cluster
func (r *PostgresWriterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.PostgresWriter{}).
		Complete(r)
}

func (r *PostgresWriterReconciler) cleanupRowFinalizerCallback(ctx context.Context, logger logr.Logger, postgresWriterObject *batchv1.PostgresWriter) error {
	// parse the table and the id of the row to delete
	id := postgresWriterObject.Namespace + "/" + postgresWriterObject.Name
	table := postgresWriterObject.Spec.Table

	// delete the row with the above 'id' from the above 'table'
	if err := r.PostgresDBClient.Delete(id, table); err != nil {
		return fmt.Errorf("error occurred while running the DELETE query on Postgres: %w", err)
	}

	// remove the cleanup-row finalizer from the postgresWriterObject
	controllerutil.RemoveFinalizer(postgresWriterObject, "finalizers.postgreswriters.batch.arshsharma.com/cleanup-row")
	if err := r.Update(ctx, postgresWriterObject); err != nil {
		return fmt.Errorf("error occurred while removing the finalizer: %w", err)
	}
	logger.Info("cleaned up the 'finalizers.postgreswriters.batch.arshsharma.com/cleanup-row' finalizer successfully")
	return nil
}
