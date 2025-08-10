package shared

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/commands/volume"
)

type VolumeInformation struct {
	VolumeID   string
	VolumeType string
	VolumeSize int32
	DeviceName string
}

type VolumeState struct {
	Current  map[string]VolumeInformation
	Desired  []v1alpha1.Storage
	Instance *types.Instance
}

type CommandAnalyzer struct {
	logger logging.Logger
}

func NewCommandAnalyzer(logger logging.Logger) *CommandAnalyzer {
	return &CommandAnalyzer{logger: logger}
}

func (a *CommandAnalyzer) BuildVolumeState(output *ec2.DescribeVolumesOutput, instance *types.Instance) *VolumeState {
	current := make(map[string]VolumeInformation)
	for _, volume := range output.Volumes {
		if volume.Attachments != nil {
			volumeID := *volume.VolumeId
			volumeDeviceName := *volume.Attachments[0].Device
			volumeSize := *volume.Size
			volumeType := string(volume.VolumeType)

			current[volumeDeviceName] = VolumeInformation{
				VolumeID:   volumeID,
				VolumeType: volumeType,
				VolumeSize: volumeSize,
				DeviceName: volumeDeviceName,
			}
		}
	}

	return &VolumeState{
		Current:  current,
		Instance: instance,
	}
}

func (a *CommandAnalyzer) AnalyzeChanges(volumeState *VolumeState) []volume.VolumeCommand {
	var cmds []volume.VolumeCommand

	for _, desired := range volumeState.Desired {
		current, exists := volumeState.Current[desired.DeviceName]
		if !exists {
			cmds = append(cmds, volume.NewVolumeCommand(
				*volumeState.Instance.InstanceId,
				*volumeState.Instance.SubnetId,
				desired.DeviceName,
				desired.InstanceDisk,
				desired.DiskSize,
			))
			continue
		}

		volumeChangeCommand := a.analyzeVolumeChanges(current, desired)
		cmds = append(cmds, volumeChangeCommand...)
	}

	volumeAttachCommands := a.analyzeDetachments(*volumeState)
	cmds = append(cmds, volumeAttachCommands...)
	return cmds
}

func (a *CommandAnalyzer) analyzeVolumeChanges(vi VolumeInformation, dv v1alpha1.Storage) []volume.VolumeCommand {
	var commands []volume.VolumeCommand
	if dv.DiskSize > vi.VolumeSize {
		updateVolumeCommand := volume.NewUpdateVolumeCommand(vi.VolumeID, dv.DiskSize)
		commands = append(commands, updateVolumeCommand)
	}

	if dv.InstanceDisk != vi.VolumeType {
		updateVolumeTypeCommand := volume.NewUpdateVolumeTypeCommand(vi.VolumeID, vi.VolumeType)
		commands = append(commands, updateVolumeTypeCommand)
	}
	return commands
}

func (a *CommandAnalyzer) analyzeDetachments(state VolumeState) []volume.VolumeCommand {
	var commands []volume.VolumeCommand

	for deviceName, current := range state.Current {
		found := false
		for _, desired := range state.Desired {
			if desired.DeviceName == deviceName {
				found = true
				break
			}
		}

		if !found {
			commands = append(commands, volume.NewDetachVolumeCommand(
				current.VolumeID,
				current.DeviceName,
				*state.Instance.InstanceId,
			))
		}
	}

	return commands
}
