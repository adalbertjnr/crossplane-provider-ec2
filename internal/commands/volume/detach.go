package volume

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type DetachVolumeCommand struct {
	BaseCommand
	VolumeId   string
	DeviceName string
	InstanceId string
}

func NewDetachVolumeCommand(volumeID, deviceName, instanceID string) *DetachVolumeCommand {
	return &DetachVolumeCommand{
		VolumeId:   volumeID,
		DeviceName: deviceName,
		InstanceId: instanceID,
	}
}

func (d *DetachVolumeCommand) Run(ctx context.Context, client *provider.EC2Client) error {
	return d.detachVolume(ctx, client, d.DeviceName, d.InstanceId, d.VolumeId)
}

func (d *DetachVolumeCommand) detachVolume(ctx context.Context, c *provider.EC2Client, deviceName, instanceId, volumeId string) error {
	_, err := c.Client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		Device:     &deviceName,
		InstanceId: &instanceId,
		VolumeId:   &volumeId,
	})

	return err
}
