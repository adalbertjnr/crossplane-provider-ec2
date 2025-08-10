package validation

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type UpdateValidator interface {
	NeedsUpdate(ctx ValidationContext) bool
	GetValidationType() string
}

type ValidationContext struct {
	Context   context.Context
	Current   *types.Instance
	Desired   *v1alpha1.InstanceConfig
	EC2Client *provider.EC2Client
}
