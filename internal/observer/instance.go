package validation

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type InstanceTypeValidator struct{}

func (v *InstanceTypeValidator) NeedsUpdate(ctx ValidationContext) bool {
	return ctx.Current.InstanceType != types.InstanceType(ctx.Desired.InstanceType)
}

func (*InstanceTypeValidator) GetValidationType() string {
	return "InstanceType"
}
