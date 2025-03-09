package cloud

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
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

func (e *EC2Client) HandleAMI(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) HandleVolume(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) HandleSecurityGroups(ctx context.Context, current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
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
