/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")

func (g *Gateway) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(g).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-raven-openyurt-io-v1alpha1-gateway,mutating=true,failurePolicy=fail,sideEffects=None,groups=raven.openyurt.io,resources=gateways,verbs=create;update,versions=v1alpha1,name=mgateway.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Gateway{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (g *Gateway) Default() {
	gatewaylog.Info("default", "name", g.Name)
	g.Spec.NodeSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCurrentGateway: g.Name,
		},
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-raven-openyurt-io-v1alpha1-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=raven.openyurt.io,resources=gateways,verbs=create;update,versions=v1alpha1,name=vgateway.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Gateway{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (g *Gateway) ValidateCreate() error {
	gatewaylog.Info("validate create", "name", g.Name)

	g.Spec.NodeSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			LabelCurrentGateway: g.Name,
		},
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (g *Gateway) ValidateUpdate(old runtime.Object) error {
	gatewaylog.Info("validate update", "name", g.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (g *Gateway) ValidateDelete() error {
	gatewaylog.Info("validate delete", "name", g.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
