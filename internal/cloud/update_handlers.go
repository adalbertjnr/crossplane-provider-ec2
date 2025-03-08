package cloud

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	property "github.com/crossplane/provider-customcomputeprovider/internal/controller/types"
)

func (e *EC2Client) HandleType(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) HandleAMI(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) HandleVolume(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}
func (e *EC2Client) HandleSecurityGroups(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
	return nil
}

func (e *EC2Client) HandleTags(current *types.Instance, desired *v1alpha1.InstanceConfig) error {
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
		tagKey, tagValue := *tag.Key, *tag.Value

		if tagKey == property.CUSTOM_PROVIDER_KEY.String() &&
			tagValue == property.CUSTOM_PROVIDER_VALUE.String() {
			continue
		}

		if _, exists := desired.InstanceTags[*tag.Key]; !exists {
			remove = append(remove, types.Tag{Key: tag.Key})
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
