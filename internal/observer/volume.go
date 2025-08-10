package validation

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/internal/shared"
	o "github.com/crossplane/provider-customcomputeprovider/internal/types"
)

type VolumeValidator struct{}

func (v *VolumeValidator) NeedsUpdate(ctx ValidationContext) bool {
	output, err := ctx.EC2Client.Client.DescribeVolumes(ctx.Context, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{Name: aws.String("attachment.instance-id"), Values: []string{*ctx.Current.InstanceId}},
		},
	})

	if err != nil {
		return false
	}

	analyzer := shared.NewCommandAnalyzer()
	state := analyzer.BuildVolumeState(output, ctx.Current)
	state.Desired = ctx.Desired.Storage
	commands := analyzer.AnalyzeChanges(state)

	return len(commands) > 0
}

func (*VolumeValidator) GetValidationType() string {
	return o.VOLUME.String()
}
