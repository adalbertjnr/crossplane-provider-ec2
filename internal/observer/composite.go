package validation

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type CompositeValidator struct {
	validators []UpdateValidator
	client     *provider.EC2Client
	logger     logging.Logger
}

type ValidationResult struct {
	HasUpdates      bool
	UpdatesRequired map[string]bool
}

func NewCompositeValidator(logger logging.Logger, client *provider.EC2Client) *CompositeValidator {
	return &CompositeValidator{
		client: client,
		logger: logger,
		validators: []UpdateValidator{
			&AMIValidator{},
			&InstanceTypeValidator{},
			&TagValidator{},
			&SecurityGroupValidator{},
			&VolumeValidator{},
		},
	}
}

func (cv *CompositeValidator) ValidateAll(ctx context.Context, currentInstance *types.Instance, desiredInstance *v1alpha1.InstanceConfig) ValidationResult {
	result := ValidationResult{UpdatesRequired: make(map[string]bool)}
	validationContext := ValidationContext{
		Context:   ctx,
		Current:   currentInstance,
		Desired:   desiredInstance,
		EC2Client: cv.client,
	}

	for _, v := range cv.validators {
		needsUpdate := v.NeedsUpdate(validationContext)
		result.UpdatesRequired[v.GetValidationType()] = needsUpdate
		if needsUpdate {
			result.HasUpdates = true
		}
		cv.logger.Info("validation result", "type", v.GetValidationType(), "needs_update", needsUpdate)
	}
	return result
}
