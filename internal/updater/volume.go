package updater

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
	"github.com/crossplane/provider-customcomputeprovider/internal/shared"
)

type VolumeUpdateOperation struct {
	BaseOperation
}

func NewVolumeOperation(logger logging.Logger) *VolumeUpdateOperation {
	return &VolumeUpdateOperation{
		BaseOperation: BaseOperation{opType: "VOLUME", logger: logger},
	}
}

func (o *VolumeUpdateOperation) Execute(ctx UpdateContext) error {
	output, err := ctx.Client.Client.DescribeVolumes(ctx.Context, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{Name: aws.String("attachment.instance-id"), Values: []string{*ctx.Current.InstanceId}},
		},
	})

	if err != nil {
		return err
	}

	analyzer := shared.NewCommandAnalyzer(o.logger)
	state := analyzer.BuildVolumeState(output, ctx.Current)
	state.Desired = ctx.Desired.Storage
	commands := analyzer.AnalyzeChanges(state)

	if len(commands) > 0 {
		turnOffChannel := make(chan struct{})

		go o.turnOff(ctx.Context, ctx.Client, turnOffChannel, *ctx.Current.InstanceId)

		<-turnOffChannel

		for _, cmd := range commands {
			if err := cmd.Run(ctx.Context, ctx.Client); err != nil {
				return err
			}
		}

		return o.turnOn(ctx.Context, ctx.Client, *ctx.Current.InstanceId)
	}

	return nil
}
func (o *VolumeUpdateOperation) turnOn(ctx context.Context, c *provider.EC2Client, instanceID string) error {
	_, err := c.Client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})

	return err
}

func (o *VolumeUpdateOperation) turnOff(ctx context.Context, c *provider.EC2Client, turnOffChannel chan<- struct{}, instanceID string) error {
	defer func() { turnOffChannel <- struct{}{} }()

	_, err := c.Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second * 10)

	for {
		<-ticker.C

		output, err := c.Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
			InstanceIds: []string{instanceID},
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
