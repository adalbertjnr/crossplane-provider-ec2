package cloud

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
)

func VolumeValidator(output *ec2.DescribeVolumesOutput, current *types.Instance, desired *v1alpha1.InstanceConfig) []VolumeCommand {
	volumeDataMap := make(map[string]volumeInformation)

	for _, volume := range output.Volumes {
		if volume.Attachments != nil {

			volumeID := *volume.VolumeId
			volumeDeviceName := *volume.Attachments[0].Device
			volumeSize := *volume.Size
			volumeType := string(volume.VolumeType)

			volumeDataMap[volumeDeviceName] = volumeInformation{
				volumeID:   volumeID,
				volumeType: volumeType,
				volumeSize: volumeSize,
			}
		}
	}

	var commands []VolumeCommand

	for _, dv := range desired.Storage {
		volume, volumeExists := volumeDataMap[dv.DeviceName]

		switch {
		case !volumeExists:
			commands = append(commands, &cvCommand{
				instanceId: *current.InstanceId,
				deviceName: dv.DeviceName,
				volumeType: dv.InstanceDisk,
				diskSize:   dv.DiskSize,
			})
			continue

		case dv.DiskSize > volume.volumeSize:
			commands = append(commands, &rvCommand{
				volumeId: volume.volumeID,
				diskSize: volume.volumeSize,
			})

		case dv.InstanceDisk != volume.volumeType:
			commands = append(commands, &cvtCommand{
				volumeId:   volume.volumeID,
				volumeType: volume.volumeType,
			})
		}
	}

	for _, cvolume := range output.Volumes {
		if _, exists := volumeDataMap[*cvolume.Attachments[0].Device]; exists {
			commands = append(commands, &dtvCommand{
				volumeId:   *cvolume.VolumeId,
				deviceName: *cvolume.Attachments[0].Device,
				instanceId: *current.InstanceId,
			})
		}
	}

	return commands
}
