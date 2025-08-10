package volume

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type UpdateVolumeCommand struct {
	BaseCommand
	VolumeID string
	DiskSize int32
}

func NewUpdateVolumeCommand(volumeID string, diskSize int32) *UpdateVolumeCommand {
	return &UpdateVolumeCommand{
		VolumeID:    volumeID,
		DiskSize:    diskSize,
		BaseCommand: BaseCommand{commandType: "Volume"},
	}
}

func (u *UpdateVolumeCommand) Run(ctx context.Context, c *provider.EC2Client) error {
	return u.updateVolumeSize(ctx, c, u.VolumeID, u.DiskSize)
}

func (u *UpdateVolumeCommand) updateVolumeSize(ctx context.Context, client *provider.EC2Client, volumeID string, volumeSize int32) error {
	_, err := client.Client.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
		VolumeId: &volumeID,
		Size:     &volumeSize,
	})
	return err
}
