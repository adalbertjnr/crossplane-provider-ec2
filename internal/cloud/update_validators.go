package cloud

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/generic"
)

func NeedsAMIUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return *current.ImageId != desired.InstanceAMI
}

func NeedsInstanceTypeUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return current.InstanceType != types.InstanceType(desired.InstanceType)
}

func NeedsTagsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	currentTags := generic.FromSliceToMap(current.Tags, func(tag types.Tag) string {
		return *tag.Key
	})

	for dk := range desired.InstanceTags {
		if _, found := currentTags[dk]; !found {
			return true
		}
	}

	return false
}

func NeedsSecurityGroupsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	currentSGIds := generic.FromSliceToMap(current.SecurityGroups, func(secId types.GroupIdentifier) string {
		return *secId.GroupId
	})

	for _, dsg := range desired.Networking.InstanceSecurityGroups {
		if _, exists := currentSGIds[dsg]; !exists {
			return true
		}
	}

	return false
}

func ResourceUpToDate(l logging.Logger, current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	amiExp := NeedsAMIUpdate(current, desired)
	typExp := NeedsInstanceTypeUpdate(current, desired)
	tagExp := NeedsTagsUpdate(current, desired)
	secExp := NeedsSecurityGroupsUpdate(current, desired)

	l.Info("check",
		"needs ami update", amiExp,
		"needs type update", typExp,
		"needs tag update", tagExp,
		"needs security groups update", secExp,
	)

	return !amiExp && !typExp && !tagExp && !secExp
}
