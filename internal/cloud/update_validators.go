package cloud

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func NeedsAMIUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return *current.ImageId != desired.InstanceAMI
}

func NeedsInstanceTypeUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return current.InstanceType != types.InstanceType(desired.InstanceType)
}

func NeedsTagsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	for _, v := range current.Tags {
		if _, found := desired.InstanceTags[*v.Key]; !found {
			return false
		}
	}

	return true
}

func NeedsSecurityGroupsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	obm := map[string]struct{}{}
	for _, csg := range current.SecurityGroups {
		if _, exists := obm[*csg.GroupId]; !exists {
			obm[*csg.GroupId] = struct{}{}
		}
	}

	for _, dsg := range desired.Networking.InstanceSecurityGroups {
		if _, exists := obm[dsg]; !exists {
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

	l.Info("resource up to date status",
		"ami update", amiExp,
		"type update", typExp,
		"tag update", tagExp,
		"security groups update", secExp,
	)

	return amiExp && typExp && tagExp && secExp
}
