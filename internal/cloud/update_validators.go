package cloud

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func InstanceAMIUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return *current.ImageId != desired.InstanceAMI
}

func InstanceTypeUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return current.InstanceType != types.InstanceType(desired.InstanceType)
}

func InstanceTagsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	for _, v := range current.Tags {
		if _, found := desired.InstanceTags[*v.Key]; !found {
			return false
		}
	}

	return true
}

func InstanceSecurityGroupsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
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
	amiE := InstanceAMIUpdate(current, desired)
	typE := InstanceTypeUpdate(current, desired)
	tagE := InstanceTagsUpdate(current, desired)
	secE := InstanceSecurityGroupsUpdate(current, desired)

	l.Info("resource up to date status",
		"ami update", amiE,
		"type update", typE,
		"tag update", tagE,
		"security groups update", secE,
	)

	return amiE && typE && tagE && secE
}
