package awspkg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func Found(ctx context.Context, e *ec2.Client, resourceName string) (bool, types.Instance, error) {
	rsp, err := e.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{resourceName}})
	if err != nil {
		slog.Error("failed to describe ec2 instance", "err", err)
		return false, types.Instance{}, fmt.Errorf("failed to describe ec2 instance: %w", err)
	}

	if len(rsp.Reservations) == 0 {
		return false, types.Instance{}, nil
	}

	return true, rsp.Reservations[0].Instances[0], nil
}

func EC2HandleInstanceType() error           { return nil }
func EC2HandleInstanceAMI() error            { return nil }
func EC2HandleInstanceTags() error           { return nil }
func EC2HandleInstanceVolume() error         { return nil }
func EC2HandleInstanceSecurityGroups() error { return nil }

type CurrentEC2Metadata struct{}

func EC2ResourceUpToDate(current types.Instance, desired *v1alpha1.InstanceConfig) bool {
	equal := false

	if current.ImageId != &desired.InstanceAMI {
		return !equal
	}

	if current.InstanceType != types.InstanceType(desired.InstanceType) {
		return !equal
	}

	for _, v := range current.Tags {
		if _, found := desired.InstanceTags[*v.Key]; !found {
			return !equal
		}
	}

	observedSecurityGroups := map[string]struct{}{}
	for _, sg := range current.SecurityGroups {
		if _, exists := observedSecurityGroups[*sg.GroupId]; !exists {
			observedSecurityGroups[*sg.GroupId] = struct{}{}
		}
	}

	for _, desiredSecurityGroup := range desired.InstanceSecurityGroups {
		if _, exists := observedSecurityGroups[desiredSecurityGroup]; !exists {
			return !equal
		}
	}

	return equal
}

func Delete(ctx context.Context, c *ec2.Client, resource v1alpha1.InstanceConfig) error {
	_, err := c.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{resource.InstanceName}})
	return err
}
