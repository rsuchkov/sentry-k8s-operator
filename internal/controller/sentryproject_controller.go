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

	"github.com/jianyuan/go-sentry/v2/sentry"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sentryv1alpha1 "github.com/rsuchkov/sentry-k8s-operator/api/v1alpha1"
)

const (
	// SentryProjectFinalizer is the name of the finalizer added to resources for deletion
	// https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#finalizers
	SentryProjectFinalizer = "sentry.io.x42r/finalizer"
)

// SentryProjectReconciler reconciles a SentryProject object
type SentryProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Sentry *sentry.Client
}

//+kubebuilder:rbac:groups=sentry.io.x42r,resources=sentryprojects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sentry.io.x42r,resources=sentryprojects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sentry.io.x42r,resources=sentryprojects/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *SentryProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	var sentryProject sentryv1alpha1.SentryProject
	log.Info("Reconciling SentryProject " + req.Name)

	if err := r.Get(ctx, req.NamespacedName, &sentryProject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Check the state of the SentryProject
	if sentryProject.Status.State == "" {
		log.Info("SentryProject state is empty, setting to Pending")
		if err := r.UpdateStatus(ctx, &sentryProject, sentryv1alpha1.Pending, ""); err != nil {
			return ctrl.Result{}, err
		}
	}
	if sentryProject.Status.State == sentryv1alpha1.Pending || sentryProject.Status.State == sentryv1alpha1.Failed {
		log.Info("SentryProject state is " + string(sentryProject.Status.State) + ", creating project")
		_, resp, err := r.CreateSentryProject(ctx, &sentryProject.Spec)
		if err != nil {
			var state sentryv1alpha1.SentryProjectCrStatus
			if resp != nil && (resp.StatusCode == 409) {
				state = sentryv1alpha1.Conflict
			} else if resp != nil && (resp.StatusCode == 404) {
				state = sentryv1alpha1.NoTeam
			} else if resp != nil && (resp.StatusCode == 403) {
				state = sentryv1alpha1.Forbiden
			} else if resp != nil && (resp.StatusCode == 400) {
				state = sentryv1alpha1.BadRequest
			} else {
				state = sentryv1alpha1.Failed
			}
			if err := r.UpdateStatus(ctx, &sentryProject, state, err.Error()); err != nil {
				return ctrl.Result{}, err
			}
		}
		// TODO: set DSN
		// sentryProject.Spec.DSN = ""
		// if err := r.Update(ctx, &sentryProject); err != nil {
		// 	return ctrl.Result{}, err
		// }
		if err := r.UpdateStatus(ctx, &sentryProject, sentryv1alpha1.Created, ""); err != nil {
			return ctrl.Result{}, err
		}
	}
	if sentryProject.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.UpdateFinalizer(ctx, &sentryProject); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if sentryProject.Status.State != sentryv1alpha1.Deleted {
			if err := r.DeleteSentryProject(ctx, &sentryProject.Spec); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.UpdateStatus(ctx, &sentryProject, sentryv1alpha1.Deleted, ""); err != nil {
				return ctrl.Result{}, err
			}
		}
		if err := r.RemoveFinalizer(ctx, &sentryProject); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *SentryProjectReconciler) CreateSentryProject(ctx context.Context, spec *sentryv1alpha1.SentryProjectSpec) (*sentry.Project, *sentry.Response, error) {
	// TODO: Add policy for creation.
	params := sentry.CreateProjectParams{Name: spec.Name, Slug: spec.Slug, Platform: spec.Platform}
	return r.Sentry.Projects.Create(ctx, spec.Organization, spec.Team, &params)
}

func (r *SentryProjectReconciler) DeleteSentryProject(ctx context.Context, spec *sentryv1alpha1.SentryProjectSpec) error {
	// TODO: Add policy for deletion.
	resp, err := r.Sentry.Projects.Delete(ctx, spec.Organization, spec.Slug)
	if err != nil && resp != nil && resp.StatusCode == 404 {
		// we don't care if the project is already deleted or doesn't exist
		// release the finalizer
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func (r *SentryProjectReconciler) UpdateStatus(ctx context.Context, sentryProject *sentryv1alpha1.SentryProject, state sentryv1alpha1.SentryProjectCrStatus, message string) error {
	sentryProject.Status.State = state
	sentryProject.Status.Message = message
	if err := r.Status().Update(ctx, sentryProject); err != nil {
		return err
	}
	return nil
}

// Add finalizer to the SentryProject resource
func (r *SentryProjectReconciler) UpdateFinalizer(ctx context.Context, sentryProject *sentryv1alpha1.SentryProject) error {
	if !containsString(sentryProject.ObjectMeta.Finalizers, SentryProjectFinalizer) {
		sentryProject.ObjectMeta.Finalizers = append(sentryProject.ObjectMeta.Finalizers, SentryProjectFinalizer)
		if err := r.Update(ctx, sentryProject); err != nil {
			return err
		}
	}
	return nil
}

// Remove finalizer from the SentryProject resource
func (r *SentryProjectReconciler) RemoveFinalizer(ctx context.Context, sentryProject *sentryv1alpha1.SentryProject) error {
	if containsString(sentryProject.ObjectMeta.Finalizers, SentryProjectFinalizer) {
		sentryProject.ObjectMeta.Finalizers = removeString(sentryProject.ObjectMeta.Finalizers, SentryProjectFinalizer)
		if err := r.Update(ctx, sentryProject); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SentryProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sentryv1alpha1.SentryProject{}).
		Complete(r)
}
