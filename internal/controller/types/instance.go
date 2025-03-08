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
