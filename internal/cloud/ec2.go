package cloud

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func (e *EC2Client) GetInstance(ctx context.Context, resourceName string) (*types.Instance, error) {
	rsp, err := e.c.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{resourceName}},
		},
	})

	if err != nil {
		slog.Error("failed to describe ec2 instance", "err", err)
		return nil, fmt.Errorf("failed to describe ec2 instance: %w", err)
	}

	if len(rsp.Reservations) == 0 {
		return nil, nil
	}

	return &rsp.Reservations[0].Instances[0], nil
}

func (e *EC2Client) Observe(ctx context.Context, resourceName string) (bool, *types.Instance, error) {
	exists := true

	instance, err := e.GetInstance(ctx, resourceName)
	if err != nil {
		return !exists, nil, err
	}

	// if the instance is shutting down or terminated the response will be != nil
	// which means we can spin up another instance with same name
	if instance == nil ||
		instance.State.Name == types.InstanceStateNameTerminated ||
		instance.State.Name == types.InstanceStateNameShuttingDown {
		return !exists, nil, nil
	}

	return exists, instance, nil
}

func (e *EC2Client) DeleteInstance(ctx context.Context, resource v1alpha1.InstanceConfig) error {
	rsp, err := e.c.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{resource.InstanceName}},
		},
	})

	if err != nil {
		slog.Error("failed to describe ec2 instance", "err", err)
		return fmt.Errorf("failed to describe ec2 instance: %w", err)
	}

	if len(rsp.Reservations) == 0 {
		slog.Info("instance not found for deletion", "err", err)
		return fmt.Errorf("instance not found for deletion: %w", err)
	}

	_, err = e.c.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{*rsp.Reservations[0].Instances[0].InstanceId},
	})

	return err
}

func (e *EC2Client) CreateInstance(ctx context.Context, resource v1alpha1.InstanceConfig) (*ec2.RunInstancesOutput, error) {
	if _, found := resource.InstanceTags["Name"]; !found {
		resource.InstanceTags["Name"] = resource.InstanceName
	}

	var computeInstanceTags []types.Tag
	for key, value := range resource.InstanceTags {
		computeInstanceTags = append(computeInstanceTags, types.Tag{Key: &key, Value: &value})
	}

	blockDeviceMapping := make([]types.BlockDeviceMapping, len(resource.Storage))

	for i, storage := range resource.Storage {
		blockDeviceMapping[i] = types.BlockDeviceMapping{
			DeviceName: &storage.DeviceName,
			Ebs: &types.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				Encrypted:           aws.Bool(true),
				VolumeType:          types.VolumeType(storage.InstanceDisk),
				VolumeSize:          &storage.DiskSize,
			},
		}
	}

	params := &ec2.RunInstancesInput{
		ImageId:             &resource.InstanceAMI,
		InstanceType:        types.InstanceType(resource.InstanceType),
		SecurityGroupIds:    resource.Networking.InstanceSecurityGroups,
		SubnetId:            &resource.Networking.SubnetID,
		MinCount:            aws.Int32(1),
		MaxCount:            aws.Int32(1),
		BlockDeviceMappings: blockDeviceMapping,

		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         computeInstanceTags,
			},
		},
	}

	return e.c.RunInstances(ctx, params)
}
