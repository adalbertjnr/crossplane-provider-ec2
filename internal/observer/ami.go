package validation

import "github.com/crossplane/provider-customcomputeprovider/internal/types"

type AMIValidator struct{}

func (v *AMIValidator) NeedsUpdate(ctx ValidationContext) bool {
	return *ctx.Current.ImageId != ctx.Desired.InstanceAMI
}

func (*AMIValidator) GetValidationType() string {
	return types.AMI.String()
}
