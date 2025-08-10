package updater

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

type SecurityGroupUpdateOperation struct {
	BaseOperation
}

func NewSecurityGroupUpdateOperation(logger logging.Logger) *SecurityGroupUpdateOperation {
	return &SecurityGroupUpdateOperation{
		BaseOperation: BaseOperation{opType: "SECURITY_GROUP", logger: logger},
	}
}
func (o *SecurityGroupUpdateOperation) Execute(ctx UpdateContext) error {
	_, err := ctx.Client.Client.DescribeSecurityGroups(ctx.Context, &ec2.DescribeSecurityGroupsInput{
		GroupIds: ctx.Desired.Networking.InstanceSecurityGroups,
	})
	if err != nil {
		return err
	}
	_, err = ctx.Client.Client.ModifyInstanceAttribute(ctx.Context, &ec2.ModifyInstanceAttributeInput{
		InstanceId: ctx.Current.InstanceId,
		Groups:     ctx.Desired.Networking.InstanceSecurityGroups,
	})

	return err
}
