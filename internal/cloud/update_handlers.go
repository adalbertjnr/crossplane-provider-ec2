package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

const errSubnetNotFound = "error subnet not found"

type VolumeCommand interface {
	Run(ctx context.Context, c *EC2Client) error
	Validate() bool
}

type cvCommand struct {
	instanceId string
	deviceName string
	volumeType string
	diskSize   int32
	subnetId   string
}

func (c *cvCommand) Validate() bool {
	return false
}

func (c *cvCommand) Run(ctx context.Context, e *EC2Client) error {
	availabilityZone, err := fetchAZ(ctx, e.Client, c.subnetId)
	if err != nil {
		return err
	}

	return e.createVolume(ctx, c.instanceId, c.deviceName, c.volumeType, availabilityZone, c.diskSize)
}

type rvCommand struct {
	volumeId string
	diskSize int32
}

func (c *rvCommand) Validate() bool {
	return false
}

func (c *rvCommand) Run(ctx context.Context, e *EC2Client) error {
	return e.updateVolumeSize(ctx, c.volumeId, c.diskSize)
}

type cvtCommand struct {
	volumeId   string
	volumeType string
}

func (c *cvtCommand) Validate() bool {
	return false
}

func (c *cvtCommand) Run(ctx context.Context, e *EC2Client) error {
	return e.updateVolumeType(ctx, c.volumeId, c.volumeType)
}

type dtvCommand struct {
	volumeId   string
	deviceName string
	instanceId string
}

func (c *dtvCommand) Validate() bool {
	return false
}

func (c *dtvCommand) Run(ctx context.Context, e *EC2Client) error {
	return e.detachVolume(ctx, c.deviceName, c.instanceId, c.volumeId)
}

func fetchAZ(ctx context.Context, c *ec2.Client, subnetID string) (string, error) {
	output, err := c.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})

	if err != nil {
		return "", err
	}

	if len(output.Subnets) > 0 {
		return *output.Subnets[0].AvailabilityZoneId, nil
	}

	return "", errors.New(errSubnetNotFound)
}

func (e *EC2Client) HandleType(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	if err := stopInstance(ctx,
		e.Client,
		current.InstanceId,
	); err != nil {
		return err
	}

	_, err := e.Client.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
		InstanceId: current.InstanceId,
		InstanceType: &types.AttributeValue{
			Value: &desired.InstanceType,
		},
	})

	if err != nil {
		return err
	}

	return startInstance(ctx, e.Client, current.InstanceId)
}

func startInstance(ctx context.Context, c *ec2.Client, instanceId *string) error {
	_, err := c.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{*instanceId},
	})
	return err
}
func stopInstance(ctx context.Context, c *ec2.Client, instanceId *string) error {
	_, err := c.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{*instanceId},
	})
	return err
}

func (e *EC2Client) HandleSecurityGroups(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	_, err := e.Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: desired.Networking.InstanceSecurityGroups,
	})

	if err != nil {
		return err
	}

	_, err = e.Client.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
		InstanceId: current.InstanceId,
		Groups:     desired.Networking.InstanceSecurityGroups,
	})

	return err
}

func (e *EC2Client) createVolume(ctx context.Context, instanceId, deviceName, volumeType, availabilityZone string, volumeSize int32) error {
	volume, err := e.Client.CreateVolume(ctx, &ec2.CreateVolumeInput{
		VolumeType:       types.VolumeType(volumeType),
		Size:             &volumeSize,
		AvailabilityZone: &availabilityZone,
	})

	if err != nil {
		return err
	}

	_, err = e.Client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device:     &deviceName,
		InstanceId: &instanceId,
		VolumeId:   volume.VolumeId,
	})

	return err
}

func (e *EC2Client) detachVolume(ctx context.Context, deviceName, instanceId, volumeId string) error {
	_, err := e.Client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		Device:     &deviceName,
		InstanceId: &instanceId,
		VolumeId:   &volumeId,
	})

	return err
}

func (e *EC2Client) updateVolumeSize(ctx context.Context, volumeId string, volumeSize int32) error {
	_, err := e.Client.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
		VolumeId: &volumeId,
		Size:     &volumeSize,
	})

	return err
}

func (e *EC2Client) updateVolumeType(ctx context.Context, volumeId, volumeType string) error {
	_, err := e.Client.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
		VolumeId:   &volumeId,
		VolumeType: types.VolumeType(volumeType),
	})

	return err
}

type volumeInformation struct {
	volumeID   string
	volumeType string
	volumeSize int32
}

func (e *EC2Client) HandleVolume(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	output, err := e.Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{Name: aws.String("attachment.instance-id"), Values: []string{*current.InstanceId}},
		},
	})

	if err != nil {
		return err
	}

	commands := VolumeValidator(output, current, desired)

	if len(commands) > 0 {
		turnOffChannel := make(chan struct{})

		go e.turnOff(ctx, turnOffChannel, *current.InstanceId)

		<-turnOffChannel

		for _, cmd := range commands {
			if err := cmd.Run(ctx, e); err != nil {
				return err
			}
		}

		return e.turnOn(ctx, *current.InstanceId)
	}

	return nil
}

func (c *EC2Client) turnOff(ctx context.Context, turnOffChannel chan<- struct{}, instanceId string) error {
	defer func() { turnOffChannel <- struct{}{} }()

	_, err := c.Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceId},
	})

	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second * 10)

	for {
		<-ticker.C

		output, err := c.Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
			InstanceIds: []string{instanceId},
		})

		if err != nil {
			return err
		}

		instanceState := output.InstanceStatuses[0].InstanceState.Name

		if instanceState != types.InstanceStateNameStopped {
			continue
		}

		break
	}

	return nil
}

func (c *EC2Client) turnOn(ctx context.Context, instanceId string) error {
	_, err := c.Client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceId},
	})

	return err
}

func (e *EC2Client) HandleTags(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	tm := make(map[string]string)

	for _, tag := range current.Tags {
		tm[*tag.Key] = *tag.Value
	}

	var update []types.Tag
	var remove []types.Tag

	for key, value := range desired.InstanceTags {
		if _, exists := tm[key]; !exists {

			update = append(update, types.Tag{Key: &key, Value: &value})
		}
	}

	for _, tag := range current.Tags {
		tagKey := *tag.Key
		if tagKey == "Name" {
			continue
		}

		if _, exists := desired.InstanceTags[*tag.Key]; !exists {
			remove = append(remove, types.Tag{Key: &tagKey})
		}
	}

	if len(update) > 0 {
		_, err := e.Client.CreateTags(context.Background(), &ec2.CreateTagsInput{
			Resources: []string{*current.InstanceId},
			Tags:      update,
		})

		if err != nil {
			return fmt.Errorf("failed to update tags: %w", err)
		}
	}

	if len(remove) > 0 {
		_, err := e.Client.DeleteTags(context.Background(), &ec2.DeleteTagsInput{
			Resources: []string{*current.InstanceId},
			Tags:      remove,
		})

		if err != nil {
			return fmt.Errorf("failed to delete tags: %w", err)
		}
	}

	return nil
}

func (e *EC2Client) HandleAMI(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
