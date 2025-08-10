package updater

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

type TagUpdateOperation struct {
	BaseOperation
}

func NewTagOperation(logger logging.Logger) *TagUpdateOperation {
	return &TagUpdateOperation{BaseOperation: BaseOperation{
		opType: "TAG",
		logger: logger,
	}}
}

func (o *TagUpdateOperation) Execute(ctx UpdateContext) error {
	tm := make(map[string]string)

	for _, tag := range ctx.Current.Tags {
		tm[*tag.Key] = *tag.Value
	}

	var update []types.Tag
	var remove []types.Tag

	for key, value := range ctx.Desired.InstanceTags {
		if _, exists := tm[key]; !exists {

			update = append(update, types.Tag{Key: &key, Value: &value})
		}
	}

	for _, tag := range ctx.Current.Tags {
		tagKey := *tag.Key
		if tagKey == "Name" {
			continue
		}

		if _, exists := ctx.Desired.InstanceTags[*tag.Key]; !exists {
			remove = append(remove, types.Tag{Key: &tagKey})
		}
	}

	if len(update) > 0 {
		_, err := ctx.Client.Client.CreateTags(context.Background(), &ec2.CreateTagsInput{
			Resources: []string{*ctx.Current.InstanceId},
			Tags:      update,
		})

		if err != nil {
			return fmt.Errorf("failed to update tags: %w", err)
		}
	}

	if len(remove) > 0 {
		_, err := ctx.Client.Client.DeleteTags(context.Background(), &ec2.DeleteTagsInput{
			Resources: []string{*ctx.Current.InstanceId},
			Tags:      remove,
		})

		if err != nil {
			return fmt.Errorf("failed to delete tags: %w", err)
		}
	}

	return nil
}
