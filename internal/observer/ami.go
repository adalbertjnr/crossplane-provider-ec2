package validation

type AMIValidator struct{}

func (v *AMIValidator) NeedsUpdate(ctx ValidationContext) bool {
	return *ctx.Current.ImageId != ctx.Desired.InstanceAMI
}

func (*AMIValidator) GetValidationType() string {
	return "AMI"
}
