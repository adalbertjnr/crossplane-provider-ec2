package cloud

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	property "github.com/crossplane/provider-customcomputeprovider/internal/controller/types"
	"github.com/crossplane/provider-customcomputeprovider/internal/generic"
)

const (
	errCustomProviderTag = "custom provider tag not found"
	errResourceNotFound  = "resource not found"
)

func (e *EC2Client) GetInstance(ctx context.Context, resourceName string) (*types.Instance, error) {
	rsp, err := e.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{resourceName}},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe ec2 instance: %w", err)
	}

	if len(rsp.Reservations) == 0 {
		return nil, nil
	}

	for _, reservation := range rsp.Reservations {
		for _, instance := range reservation.Instances {
			if managedInstance, err := getManagedResource(instance); err == nil {
				return &managedInstance, nil
			}
		}
	}

	return nil, errors.Wrap(err, errResourceNotFound)
}

func getManagedResource(instance types.Instance) (types.Instance, error) {
	resourceState := instance.State.Name

	tm := generic.FromSliceToMapWithValues(instance.Tags, func(tag types.Tag) (string, string) {
		return *tag.Key, *tag.Value
	})

	if v, found := tm[property.CUSTOM_PROVIDER_KEY.String()]; !found && v != property.CUSTOM_PROVIDER_VALUE.String() {
		return types.Instance{}, errors.New(errCustomProviderTag)
	}

	switch resourceState {
	case types.InstanceStateNameStopped,
		types.InstanceStateNameRunning,
		types.InstanceStateNamePending:
		return instance, nil
	default:
		return types.Instance{}, errors.New(errResourceNotFound)
	}
}

func (e *EC2Client) Observe(ctx context.Context, resourceName string) (bool, *types.Instance, error) {
	exists := true

	instance, err := e.GetInstance(ctx, resourceName)
	if err != nil {
		if err.Error() == errResourceNotFound {
			return !exists, nil, nil
		}
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
	rsp, err := e.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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

	_, err = e.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
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

	cpManagedKey := property.CUSTOM_PROVIDER_KEY.String()
	cpManagedValue := property.CUSTOM_PROVIDER_VALUE.String()

	computeInstanceTags = append(computeInstanceTags, types.Tag{Key: &cpManagedKey, Value: &cpManagedValue})

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
