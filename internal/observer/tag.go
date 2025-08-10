package validation

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	o "github.com/crossplane/provider-customcomputeprovider/internal/types"
	"github.com/crossplane/provider-customcomputeprovider/pkg/generic"
)

type TagValidator struct{}

func (v *TagValidator) compareTagMaps(current, desired map[string]string) bool {
	for k, v := range desired {
		if cv, exists := current[k]; !exists || cv != v {
			return true
		}
	}

	for k := range current {
		if k == "Name" {
			continue
		}
		if _, exists := desired[k]; !exists {
			return true
		}
	}

	return false
}

func (v *TagValidator) NeedsUpdate(ctx ValidationContext) bool {
	currentTags := generic.FromSliceToMapWithValues(ctx.Current.Tags,
		func(tag types.Tag) (string, string) {
			return *tag.Key, *tag.Value
		})

	return v.compareTagMaps(currentTags, ctx.Desired.InstanceTags)
}

func (*TagValidator) GetValidationType() string {
	return o.TAGS.String()
}
