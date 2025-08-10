package updater

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

type TypeUpdateOperation struct {
	BaseOperation
}

func NewTypeUpdateOperation(logger logging.Logger) *TypeUpdateOperation {
	return &TypeUpdateOperation{BaseOperation: BaseOperation{opType: "TYPE", logger: logger}}
}

func (u *TypeUpdateOperation) Execute(ctx UpdateContext) error {
	if err := stopInstance(ctx.Context,
		ctx.Client.Client,
		ctx.Current.InstanceId,
	); err != nil {
		return err
	}

	_, err := ctx.Client.Client.ModifyInstanceAttribute(ctx.Context, &ec2.ModifyInstanceAttributeInput{
		InstanceId: ctx.Current.InstanceId,
		InstanceType: &types.AttributeValue{
			Value: &ctx.Desired.InstanceType,
		},
	})

	if err != nil {
		return err
	}
	return startInstance(ctx.Context, ctx.Client.Client, ctx.Current.InstanceId)
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
