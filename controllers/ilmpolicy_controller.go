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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	elasticsearchv1alpha1 "ilm-operator/api/v1alpha1"
)

// ILMPolicyReconciler reconciles a ILMPolicy object
type ILMPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type ElasticsearchClusterClient struct {
	ElasticsearchEndpoint string
	AuthorizationHeader   string
}

func (r *ILMPolicyReconciler) initElasticsearchClusterClient(ctx context.Context, policy *elasticsearchv1alpha1.ILMPolicy) (*ElasticsearchClusterClient, error) {
	log := log.FromContext(ctx)
	endpoint, err := r.getElasticsearchEndpoint(policy)
	if err != nil {
		log.Error(err, "Error while getting the elasticsearch endpoint")
		return nil, err
	}
	authorizationHeader, err := r.getElasticsearchAuthorizationHeader(ctx, policy)
	if err != nil {
		log.Error(err, "Error while getting the authorization endpoint")
		return nil, err
	}

	client := &ElasticsearchClusterClient{
		ElasticsearchEndpoint: endpoint,
		AuthorizationHeader:   authorizationHeader,
	}

	return client, nil
}

func (r *ILMPolicyReconciler) getElasticsearchEndpoint(policy *elasticsearchv1alpha1.ILMPolicy) (string, error) {
	elasticsearchService := fmt.Sprintf("%s-es-http.%s.svc.cluster.local", policy.Spec.ElasticsearchCluster, policy.Namespace)

	return elasticsearchService, nil
}

func (r *ILMPolicyReconciler) getElasticsearchAuthorizationHeader(ctx context.Context, policy *elasticsearchv1alpha1.ILMPolicy) (string, error) {
	log := log.FromContext(ctx)
	elasticsearchCredentialsName := fmt.Sprintf("%s-es-elastic-user", policy.Spec.ElasticsearchCluster)

	secret := &corev1.Secret{}
	log.Info(fmt.Sprintf("before fetching secret: %s", secret))
	secretKey := client.ObjectKey{
		Namespace: policy.Namespace,
		Name:      elasticsearchCredentialsName,
	}
	r.Get(ctx, secretKey, secret)

	elasticUserPassword := secret.Data["elastic"]
	log.Info(fmt.Sprintf("elastic user password: %s", elasticUserPassword))

	elasticUserPasswordEncoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "elastic", elasticUserPassword)))

	return elasticUserPasswordEncoded, nil
}

//+kubebuilder:rbac:groups=elasticsearch.my.domain,resources=ilmpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearch.my.domain,resources=ilmpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearch.my.domain,resources=ilmpolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ILMPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ILMPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	policy := &elasticsearchv1alpha1.ILMPolicy{}
	err := r.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		log.Error(err, "Unable to fetch the ILM Policy")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	elasticsearchClient, err := r.initElasticsearchClusterClient(ctx, policy)
	if err != nil {
		log.Error(err, "Error while retrieving the elasticsearch client")
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("Retrieved ILM Policy, %+v", policy))

	myFinalizerName := "batch.tutorial.kubebuilder.io/finalizer"
	// using finalizers https://book.kubebuilder.io/reference/using-finalizers.html
	if policy.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(policy.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(policy, myFinalizerName)
			if err != r.Update(ctx, policy) {
				return ctrl.Result{}, nil
			}
		}
	} else {
		if containsString(policy.GetFinalizers(), myFinalizerName) {
			if err := r.deleteExternalResources(ctx, httpClient, policy, elasticsearchClient); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(policy, myFinalizerName)
			if err := r.Update(ctx, policy); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// getting the desired state
	elasticsearchILMPolicyEndpoint := fmt.Sprintf("https://%s:9200/_ilm/policy", elasticsearchClient.ElasticsearchEndpoint)

	log.Info(fmt.Sprintf("Elasticsearch ILM Policy endpoint: %s", elasticsearchILMPolicyEndpoint))
	log.Info(fmt.Sprintf("Authorization Header: %s", elasticsearchClient.AuthorizationHeader))

	// Create/update the ILM Policy. It's always a PUT request.
	log.Info("Creating/Updating the ILM policy")
	policyBody := policy.Spec.Body

	// check JSON for http methods https://riptutorial.com/go/example/27703/put-request-of-json-object
	// input validation
	policyRequest, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s", elasticsearchILMPolicyEndpoint, req.Name), bytes.NewBufferString(policyBody))
	if err != nil {
		log.Error(err, "Error while creating the ILM policy create/update HTTP request")
		return ctrl.Result{}, nil
	}

	policyRequest.Header.Set("Content-Type", "application/json")
	policyRequest.Header.Add("Authorization", fmt.Sprintf("Basic %s", elasticsearchClient.AuthorizationHeader))

	resp, err := httpClient.Do(policyRequest)
	log.Info(fmt.Sprintf("Policy creation response: %+v", resp))
	if err != nil {
		log.Error(err, "Error while sending the HTTP request to create/update the policy")
		return ctrl.Result{}, nil
	}

	if resp.StatusCode < 200 && resp.StatusCode >= 300 {
		log.Info(fmt.Sprintf("non-200 return code: %d, status: %+v", resp.StatusCode, resp.Status))
	}

	log.Info("End of the reconcile loop")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ILMPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchv1alpha1.ILMPolicy{}).
		Complete(r)
}

// func (r *ILMPolicyReconciler) deleteExternalResources(ctx context.Context, httpClient *http.Client, policy *elasticsearchv1alpha1.ILMPolicy, elasticsearchILMPolicyEndpoint string) error {
func (r *ILMPolicyReconciler) deleteExternalResources(ctx context.Context, httpClient *http.Client, policy *elasticsearchv1alpha1.ILMPolicy, client *ElasticsearchClusterClient) error {
	//
	// delete any external resources associated with the cronJob
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object.
	// log.Info(fmt.Sprintf("Deleting external resources %+v", policy))
	fmt.Printf("Deleting external resources %+v", policy)
	log := log.FromContext(ctx)

	log.Info("Deleting the ILM Policy")

	elasticsearchILMPolicyEndpoint := fmt.Sprintf("https://%s:9200/_ilm/policy", client.ElasticsearchEndpoint)

	deleteRequest, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s", elasticsearchILMPolicyEndpoint, policy.Name), nil)
	if err != nil {
		log.Error(err, "Error while creating the ILM policy HTTP deletion request")
		return err
	}

	deleteRequest.Header.Add("Authorization", fmt.Sprintf("Basic %s", client.AuthorizationHeader))

	deleteResponse, err := httpClient.Do(deleteRequest)
	if err != nil {
		log.Error(err, "Error while sending the ILM policy HTTP deletion request")
		return err
	}
	log.Info(fmt.Sprintf("delete response: %+v", deleteResponse))

	if deleteResponse.StatusCode < 200 && deleteResponse.StatusCode >= 300 {
		log.Info(fmt.Sprintf("non-200 return code: %d", deleteResponse.StatusCode))
		log.Info(fmt.Sprintf("status: %+v", deleteResponse.Status))
	}

	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
