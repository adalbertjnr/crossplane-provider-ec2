package updater

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

type NameUpdateOperation struct {
	BaseOperation
}

func NewNameOperation(logger logging.Logger) *NameUpdateOperation {
	return &NameUpdateOperation{
		BaseOperation: BaseOperation{opType: "NAME", logger: logger},
	}
}

func (o *NameUpdateOperation) Execute(ctx UpdateContext) error {
	patchName := types.Tag{Key: aws.String("Name"), Value: &ctx.Desired.InstanceName}
	_, err := ctx.Client.Client.CreateTags(ctx.Context, &ec2.CreateTagsInput{
		Resources: []string{*ctx.Current.InstanceId},
		Tags:      []types.Tag{patchName},
	})
	return err
}
