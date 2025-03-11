package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	volumeproperty "github.com/crossplane/provider-customcomputeprovider/internal/controller/types"
)

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

func (e *EC2Client) HandleVolume(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	output, err := e.Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{Name: aws.String("attachment.instance-id"), Values: []string{*current.InstanceId}},
		},
	})

	if err != nil {
		return err
	}

	updateFuncs := map[volumeproperty.VolumeProperty]func(c *types.Instance) error{
		volumeproperty.VOLUME_UPGRADE: func(c *types.Instance) error {

			return nil
		},

		volumeproperty.VOLUME_TYPE: func(c *types.Instance) error {

			return nil
		},
	}

	var updateKeys []volumeproperty.VolumeProperty

	for i, volume := range output.Volumes {
		volumeSize := *volume.Size
		volumeType := string(volume.VolumeType)

		if desired.Storage[i].DiskSize != volumeSize && desired.Storage[i].DiskSize > volumeSize {
			updateKeys = append(updateKeys, volumeproperty.VOLUME_UPGRADE)
		}

		if desired.Storage[i].InstanceDisk != volumeType {
			updateKeys = append(updateKeys, volumeproperty.VOLUME_TYPE)
		}
	}

	turnOffChannel := make(chan struct{})

	if len(updateKeys) > 0 {
		go e.turnOff(ctx, turnOffChannel, *current.InstanceId)
	}

	<-turnOffChannel

	for _, updateKey := range updateKeys {
		if updateFunc, appended := updateFuncs[updateKey]; appended {

			if err := updateFunc(current); err != nil {
				return err
			}
		}
	}

	return e.turnOn(ctx, *current.InstanceId)
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
