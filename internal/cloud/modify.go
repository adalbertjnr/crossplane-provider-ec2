package cloud

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
		if _, exists := desired.InstanceTags[*tag.Key]; !exists {
			remove = append(remove, types.Tag{Key: tag.Key})
		}
	}

	if len(update) > 0 {
		_, err := e.c.CreateTags(context.Background(), &ec2.CreateTagsInput{
			Resources: []string{*current.InstanceId},
			Tags:      update,
		})

		if err != nil {
			return fmt.Errorf("failed to update tags: %w", err)
		}
	}

	if len(remove) > 0 {
		_, err := e.c.DeleteTags(context.Background(), &ec2.DeleteTagsInput{
			Resources: []string{*current.InstanceId},
			Tags:      update,
		})

		if err != nil {
			return fmt.Errorf("failed to delete tags: %w", err)
		}
	}

	return nil
}
