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
	"time"

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
	// Checking the state of the SentryProject
	if !sentryProject.IsProjectCreated() {
		// if the project is not created, create the project
		log.Info("Creating sentry project: " + sentryProject.Spec.Slug)
		if err := r.CreateSentryProject(ctx, &sentryProject); err != nil {
			msg := "Failed to create project: " + err.Error()
			if err := r.UpdateReadyCondition(ctx, &sentryProject, sentryv1alpha1.False, msg); err != nil {
				return ctrl.Result{}, err
			}
			log.Error(err, msg)
		}
		sentryProject.Status.Slug = sentryProject.Spec.Slug
		if err := r.UpdateReadyCondition(ctx, &sentryProject, sentryv1alpha1.True, ""); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.UpdateFinalizer(ctx, &sentryProject); err != nil {
			return ctrl.Result{}, err
		}
	} else if !sentryProject.ObjectMeta.DeletionTimestamp.IsZero() {
		// if the project is being deleted, delete the project
		log.Info("Deleting sentry project: " + sentryProject.Spec.Slug)
		if err := r.DeleteSentryProject(ctx, &sentryProject); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.RemoveFinalizer(ctx, &sentryProject); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// if the project is created and the condition is true, update the project
		log.Info("Updating sentry project: " + sentryProject.Spec.Slug)
		if err := r.UpdateSentryProject(ctx, &sentryProject); err != nil {
			msg := "Failed to update project: " + err.Error()
			if err := r.UpdateReadyCondition(ctx, &sentryProject, sentryv1alpha1.False, msg); err != nil {
				return ctrl.Result{}, err
			}
			log.Error(err, msg)
		} else {
			sentryProject.Status.Slug = sentryProject.Spec.Slug
			if err := r.UpdateReadyCondition(ctx, &sentryProject, sentryv1alpha1.True, ""); err != nil {
				return ctrl.Result{}, err
			}
		}

	}
	return ctrl.Result{}, nil
}

func (r *SentryProjectReconciler) GetSentryProject(ctx context.Context, organization, slug string) (*sentry.Project, error) {
	pr, _, err := r.Sentry.Projects.Get(ctx, organization, slug)
	return pr, err
}

func (r *SentryProjectReconciler) CreateSentryProject(ctx context.Context, proj *sentryv1alpha1.SentryProject) error {
	spec := proj.Spec
	params := sentry.CreateProjectParams{Name: spec.Name, Slug: spec.Slug, Platform: spec.Platform}
	_, resp, err := r.Sentry.Projects.Create(ctx, spec.Organization, spec.Team, &params)
	if err != nil {
		if resp != nil && resp.StatusCode == 409 {
			if spec.ConflictPolicy == sentryv1alpha1.Ignore {
				return nil
			} else if spec.ConflictPolicy == sentryv1alpha1.Update {
				if err := r.UpdateSentryProject(ctx, proj); err != nil {
					return err
				}
			}
		}
		return err
	}
	return nil
}

func (r *SentryProjectReconciler) UpdateSentryProject(ctx context.Context, proj *sentryv1alpha1.SentryProject) error {
	// TODO: Handle the case with Organization change
	spec := proj.Spec
	params := sentry.UpdateProjectParams{Name: spec.Name, Slug: spec.Slug, Platform: spec.Platform}
	_, _, err := r.Sentry.Projects.Update(ctx, spec.Organization, proj.Status.Slug, &params)
	if err != nil {
		return err
	}
	return nil
}

func (r *SentryProjectReconciler) DeleteSentryProject(ctx context.Context, proj *sentryv1alpha1.SentryProject) error {
	// TODO: Add policy for deletion.
	spec := proj.Spec
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

func (r *SentryProjectReconciler) UpdateReadyCondition(ctx context.Context, sentryProject *sentryv1alpha1.SentryProject, status sentryv1alpha1.ConditionStatus, message string) error {
	condition := sentryv1alpha1.Condition{Type: sentryv1alpha1.Ready, Status: status, Message: message}
	return r.UpdateCondition(ctx, sentryProject, condition)
}

func (r *SentryProjectReconciler) UpdateCondition(ctx context.Context, sentryProject *sentryv1alpha1.SentryProject, condition sentryv1alpha1.Condition) error {
	condition.LastTransitionTime = time.Now().Format(time.RFC3339)
	// Find if the condition already exists
	exists := false
	for i, c := range sentryProject.Status.Conditions {
		if c.Type == condition.Type {
			sentryProject.Status.Conditions[i] = condition
			exists = true
		}
	}
	if !exists {
		sentryProject.Status.Conditions = append(sentryProject.Status.Conditions, condition)
	}
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
