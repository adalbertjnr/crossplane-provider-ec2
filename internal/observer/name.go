package validation

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	o "github.com/crossplane/provider-customcomputeprovider/internal/types"
	"github.com/crossplane/provider-customcomputeprovider/pkg/generic"
)

type NameValidator struct{}

func (v *NameValidator) NeedsUpdate(ctx ValidationContext) bool {
	currentInstanceTags := generic.FromSliceToMapWithValues(ctx.Current.Tags,
		func(tag types.Tag) (string, string) { return *tag.Key, *tag.Value },
	)

	if currentName, found := currentInstanceTags["Name"]; found {
		return ctx.Desired.InstanceName != currentName
	}

	return false
}

func (*NameValidator) GetValidationType() string {
	return o.NAME.String()
}
