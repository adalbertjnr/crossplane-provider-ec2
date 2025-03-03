package awspkg

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func (e *EC2Client) EC2HandleInstanceType(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) EC2HandleInstanceAMI(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) EC2HandleInstanceVolume(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) EC2HandleInstanceSecurityGroups(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}

func (e *EC2Client) EC2HandleInstanceTags(current *types.Instance, desired *v1alpha1.InstanceConfig) error {

	return nil
}
