/*
Copyright 2022 The Crossplane Authors.

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

package compute

import (
	"context"
	"encoding/json"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-customcomputeprovider/apis/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/features"
	validation "github.com/crossplane/provider-customcomputeprovider/internal/observer"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
	"github.com/crossplane/provider-customcomputeprovider/internal/updater"
)

const (
	errNotCompute   = "managed resource is not a Compute custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
	errAwsClient = "cannot create aws client"
)

// A NoOpService does nothing.
type NoOpService struct{}

var (
	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
)

// Setup adds a controller that reconciles Compute managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ComputeGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ComputeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newNoOpService,
			logger:       o.Logger,
		},
		),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Compute{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (interface{}, error)
	logger       logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return nil, errors.New(errNotCompute)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	credentialsClient := func(creds []byte) (interface{}, error) {
		var awsCredentials struct {
			AccessKeyID     string `json:"access_key_id"`
			SecretAccessKey string `json:"secret_access_key"`
		}

		if err := json.Unmarshal(creds, &awsCredentials); err != nil {
			return nil, err
		}

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cr.Spec.ForProvider.AWSConfig.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				awsCredentials.AccessKeyID,
				awsCredentials.SecretAccessKey,
				"",
			)),
		)

		if err != nil {
			return nil, err
		}

		client := ec2.NewFromConfig(cfg)
		return &provider.EC2Client{Client: client}, nil

	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return &external{}, errors.Wrap(err, errGetCreds)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := credentialsClient(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, logger: c.logger, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service interface{}
	logger  logging.Logger
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCompute)
	}

	log := c.logger.WithValues(
		"action", "observe",
		"resource", cr.Name,
		"instanceName", cr.Spec.ForProvider.InstanceConfig.InstanceName,
	)

	client, err := clientSelector(ctx, c, cr.Spec.ForProvider.AWSConfig.Region)
	if err != nil {
		log.Info("failed to get EC2 client", "error", err)
		return managed.ExternalObservation{}, err
	}

	resourceConfig := cr.Spec.ForProvider.InstanceConfig
	if cr.Status.AtProvider.InstanceID == "" {
		log.Info("instance ID not found, resource does not exist")
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	log = log.WithValues("instanceID", cr.Status.AtProvider.InstanceID)

	resourceFound, currentResource, err := client.Observe(ctx, cr, resourceConfig.InstanceName)
	if err != nil {
		log.Info("failed to observe resource", "error", err)
		return managed.ExternalObservation{}, err
	}

	if !resourceFound {
		log.Info("resource not found", "the program will initialize the creation",
			"region", cr.Spec.ForProvider.AWSConfig.Region,
			"config", resourceConfig,
		)
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	log = log.WithValues(
		"state", currentResource.State.Name,
		"type", currentResource.InstanceType,
		"launchTime", currentResource.LaunchTime,
	)

	validators := validation.NewCompositeValidator(c.logger, client)
	validationResults := validators.ValidateAll(ctx, currentResource, &resourceConfig)

	if validationResults.HasUpdates {
		log.Info("resource needs update",
			"updates", validationResults.UpdatesRequired,
			"current", map[string]interface{}{
				"type":           currentResource.InstanceType,
				"ami":            *currentResource.ImageId,
				"state":          currentResource.State.Name,
				"securityGroups": currentResource.SecurityGroups,
				"tags":           processTags(currentResource.Tags),
			},
			"desired", map[string]interface{}{
				"type":           resourceConfig.InstanceType,
				"ami":            resourceConfig.InstanceAMI,
				"securityGroups": resourceConfig.Networking.InstanceSecurityGroups,
				"tags":           resourceConfig.InstanceTags,
			},
		)
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	log.Info("resource is up to date",
		"currentState", map[string]interface{}{
			"type":           currentResource.InstanceType,
			"ami":            *currentResource.ImageId,
			"state":          currentResource.State.Name,
			"securityGroups": currentResource.SecurityGroups,
			"tags":           processTags(currentResource.Tags),
		},
	)

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCompute)
	}

	log := c.logger.WithValues(
		"action", "create",
		"resource", cr.Name,
		"region", cr.Spec.ForProvider.AWSConfig.Region,
	)

	cc, err := clientSelector(ctx, c, cr.Spec.ForProvider.AWSConfig.Region)
	if err != nil {
		log.Debug("failed to get ec2 client", "error", err, "region", cr.Spec.ForProvider.AWSConfig.Region)
		return managed.ExternalCreation{}, err
	}

	resourceConfig := cr.Spec.ForProvider.InstanceConfig

	log.Info("initiating instance creation",
		"config", map[string]interface{}{
			"name":           resourceConfig.InstanceName,
			"type":           resourceConfig.InstanceType,
			"ami":            resourceConfig.InstanceAMI,
			"subnet":         resourceConfig.Networking.SubnetID,
			"securityGroups": resourceConfig.Networking.InstanceSecurityGroups,
			"storage":        len(resourceConfig.Storage),
			"tags":           resourceConfig.InstanceTags,
		},
	)

	runOutput, err := cc.CreateInstance(ctx, resourceConfig)
	if err != nil {
		log.Debug("failed to create instance",
			"error", err,
			"config", resourceConfig,
		)
		return managed.ExternalCreation{}, err
	}

	instanceID := *runOutput.Instances[0].InstanceId
	instanceStatus := string(runOutput.Instances[0].State.Name)

	patchCR := cr.DeepCopy()
	patchCR.Status.AtProvider = v1alpha1.ComputeObservation{
		InstanceID: instanceID,
		State:      instanceStatus,
	}

	mergeSource := client.MergeFrom(cr)

	if err := c.kube.Status().Patch(ctx, patchCR, mergeSource); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to update compute status after resource creation")
	}

	log.Info("instance created successfully",
		"details", map[string]interface{}{
			"instanceID": instanceID,
			"status":     instanceStatus,
			"timestamp":  time.Now().UTC(),
			"name":       resourceConfig.InstanceName,
			"type":       resourceConfig.InstanceType,
			"networking": map[string]interface{}{
				"subnet":         resourceConfig.Networking.SubnetID,
				"securityGroups": resourceConfig.Networking.InstanceSecurityGroups,
			},
			"storage": map[string]interface{}{
				"volumes": len(resourceConfig.Storage),
				"config":  resourceConfig.Storage,
			},
		},
	)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCompute)
	}

	desiredConfig := cr.Spec.ForProvider.InstanceConfig
	client, err := clientSelector(ctx, c, cr.Spec.ForProvider.AWSConfig.Region)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	instanceID := cr.Status.AtProvider.InstanceID

	currentConfig, err := client.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	validator := validation.NewCompositeValidator(c.logger, client)
	validationResult := validator.ValidateAll(ctx, currentConfig, &desiredConfig)

	updateCtx := updater.UpdateContext{
		Context: ctx,
		Current: currentConfig,
		Desired: &desiredConfig,
		Client:  client,
		Logger:  c.logger,
	}

	orchestrator := updater.NewUpdateOrchestrator(c.logger)
	if err := orchestrator.ExecuteUpdates(updateCtx, validationResult.UpdatesRequired); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Compute)
	if !ok {
		return errors.New(errNotCompute)
	}

	log := c.logger.WithValues(
		"action", "delete",
		"resource", cr.Name,
		"instanceID", cr.Status.AtProvider.InstanceID,
		"instanceName", cr.Spec.ForProvider.InstanceConfig.InstanceName,
		"region", cr.Spec.ForProvider.AWSConfig.Region,
	)

	resourceConfig := cr.Spec.ForProvider.InstanceConfig
	client, err := clientSelector(ctx, c, cr.Spec.ForProvider.AWSConfig.Region)
	if err != nil {
		return err
	}

	instanceID := cr.Status.AtProvider.InstanceID

	log.Info("successfully initiated instance deletion",
		"details", map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"config": map[string]interface{}{
				"subnet":         resourceConfig.Networking.SubnetID,
				"securityGroups": resourceConfig.Networking.InstanceSecurityGroups,
			},
		},
	)

	return client.DeleteInstanceByID(ctx, instanceID)
}

func processTags(tags []ec2types.Tag) map[string]string {
	m := make(map[string]string, len(tags))

	for _, v := range tags {
		if _, exists := m[*v.Key]; !exists {
			m[*v.Key] = *v.Value
		}
	}

	return m
}

func clientSelector(ctx context.Context, c *external, region string) (*provider.EC2Client, error) {
	var client *provider.EC2Client

	cc, ok := c.service.(*provider.EC2Client)
	if ok {
		client = cc
	} else {
		cfg, err := provider.AWSClientConnector(ctx)(region)
		if err != nil {
			return nil, errors.New(errAwsClient)
		}
		cc := provider.NewEC2Client(cfg)
		client = cc
	}

	return client, nil
}
