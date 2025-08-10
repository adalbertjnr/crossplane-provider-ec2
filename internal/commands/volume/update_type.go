package volume

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type UpdateVolumeTypeCommand struct {
	BaseCommand
	VolumeID   string
	VolumeType string
}

func NewUpdateVolumeTypeCommand(volumeID, volumeType string) *UpdateVolumeTypeCommand {
	return &UpdateVolumeTypeCommand{
		BaseCommand: BaseCommand{commandType: "VolumeType"},
		VolumeID:    volumeID,
		VolumeType:  volumeType,
	}
}

func (u *UpdateVolumeTypeCommand) Run(ctx context.Context, c *provider.EC2Client) error {
	return u.updateVolumeType(ctx, c, u.VolumeID, u.VolumeType)
}

func (u *UpdateVolumeTypeCommand) updateVolumeType(ctx context.Context, c *provider.EC2Client, volumeID, volumeType string) error {
	_, err := c.Client.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
		VolumeId:   &volumeID,
		VolumeType: types.VolumeType(volumeType),
	})

	return err
}
