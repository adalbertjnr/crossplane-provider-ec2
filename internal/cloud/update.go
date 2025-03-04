package cloud

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func InstanceAMIUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return current.ImageId != &desired.InstanceAMI
}

func InstanceTypeUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return current.InstanceType != types.InstanceType(desired.InstanceType)
}

func InstanceTagsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	for _, v := range current.Tags {
		if _, found := desired.InstanceTags[*v.Key]; !found {
			return true
		}
	}

	return false
}

func InstanceSecurityGroupsUpdate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	observedSecurityGroups := map[string]struct{}{}
	for _, sg := range current.SecurityGroups {
		if _, exists := observedSecurityGroups[*sg.GroupId]; !exists {
			observedSecurityGroups[*sg.GroupId] = struct{}{}
		}
	}

	for _, desiredSecurityGroup := range desired.Networking.InstanceSecurityGroups {
		if _, exists := observedSecurityGroups[desiredSecurityGroup]; !exists {
			return true
		}
	}

	return false
}

func EC2ResourceUpToDate(current *types.Instance, desired *v1alpha1.InstanceConfig) bool {
	return !InstanceAMIUpdate(current, desired) ||
		!InstanceTypeUpdate(current, desired) ||
		!InstanceTagsUpdate(current, desired) ||
		!InstanceSecurityGroupsUpdate(current, desired)
}
