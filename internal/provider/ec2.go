package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

var errInstanceNotFound = errors.New("error instance not found")

func (e *EC2Client) GetInstanceByID(ctx context.Context, instanceID string) (*types.Instance, error) {
	instance, err := e.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe ec2 instance %s: %w", instanceID, err)
	}

	if len(instance.Reservations) == 0 {
		return nil, errInstanceNotFound
	}

	return &instance.Reservations[0].Instances[0], nil
}

func (e *EC2Client) Observe(ctx context.Context, cr *v1alpha1.Compute, resourceName string) (bool, *types.Instance, error) {
	exists := true

	instanceID := cr.Status.AtProvider.InstanceID

	instance, err := e.GetInstanceByID(ctx, instanceID)
	if err != nil {
		if errors.Is(err, errInstanceNotFound) {
			return !exists, nil, nil
		}

		return !false, nil, err
	}

	return exists, instance, nil
}

func (e *EC2Client) DeleteInstanceByID(ctx context.Context, instanceID string) error {
	output, err := e.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		slog.Error("failed to describe ec2 instance", "err", err)
		return fmt.Errorf("failed to describe ec2 instance: %w", err)
	}

	if len(output.Reservations) == 0 {
		slog.Info("instance not found for deletion", "err", err)
		return fmt.Errorf("instance not found for deletion: %w", err)
	}

	_, err = e.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{*output.Reservations[0].Instances[0].InstanceId},
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

	return e.Client.RunInstances(ctx, params)
}
