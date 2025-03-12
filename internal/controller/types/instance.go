package types

type Property string

const (
	SECURITY_GROUPS Property = "SecurityGroups"
	TAGS            Property = "Tags"
	INSTANCE_TYPE   Property = "InstanceType"
	AMI             Property = "AMI"
	VOLUME          Property = "Volumes"
)

func (p Property) String() string {
	return string(p)
}

type VolumeProperty string

const (
	VOLUME_UPGRADE    VolumeProperty = "VolumeUpgrade"
	VOLUME_TYPE       VolumeProperty = "VolumeType"
	VOLUME_ATTACHMENT VolumeProperty = "VolumeAttachment"
	VOLUME_CREATE     VolumeProperty = "VolumeCreate"
)
