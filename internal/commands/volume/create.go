package volume

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type CreateVolumeCommand struct {
	BaseCommand
	InstanceId string
	DeviceName string
	VolumeType string
	DiskSize   int32
	SubnetId   string
}

func NewVolumeCommand(instanceID, subnetID, deviceName, volumeType string, diskSize int32) *CreateVolumeCommand {
	return &CreateVolumeCommand{
		BaseCommand: BaseCommand{commandType: "CreateVolume"},
		InstanceId:  instanceID,
		DeviceName:  deviceName,
		VolumeType:  volumeType,
		DiskSize:    diskSize,
		SubnetId:    subnetID,
	}
}

func (c *CreateVolumeCommand) Run(ctx context.Context, client *provider.EC2Client) error {
	availabilityZone, err := getAvailabilityZone(ctx, client.Client, c.SubnetId)
	if err != nil {
		return err
	}

	return c.createVolume(ctx, client, c.InstanceId, c.DeviceName, c.VolumeType, availabilityZone, c.DiskSize)
}

func getAvailabilityZone(ctx context.Context, c *ec2.Client, subnetID string) (string, error) {
	output, err := c.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})

	if err != nil {
		return "", err
	}

	if len(output.Subnets) > 0 {
		return *output.Subnets[0].AvailabilityZoneId, nil
	}

	return "", errors.New("subnet not found")
}

func (e *CreateVolumeCommand) createVolume(ctx context.Context, client *provider.EC2Client, instanceId, deviceName, volumeType, availabilityZone string, volumeSize int32) error {
	volume, err := client.Client.CreateVolume(ctx, &ec2.CreateVolumeInput{
		VolumeType:       types.VolumeType(volumeType),
		Size:             &volumeSize,
		AvailabilityZone: &availabilityZone,
	})

	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second * 10)

	for {
		<-ticker.C

		output, err := client.Client.DescribeVolumeStatus(ctx, &ec2.DescribeVolumeStatusInput{
			VolumeIds: []string{*volume.VolumeId},
		})

		if err != nil {
			return err
		}

		status := output.VolumeStatuses[0].VolumeStatus.Status
		desiredState := types.VolumeStatusInfoStatus(types.StateAvailable)

		if status != desiredState {
			continue
		}

		break
	}

	_, err = client.Client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device:     &deviceName,
		InstanceId: &instanceId,
		VolumeId:   volume.VolumeId,
	})

	return err
}
