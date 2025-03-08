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

type CustomProviderTags string

const (
	CUSTOM_PROVIDER_KEY   CustomProviderTags = "ManagedBy"
	CUSTOM_PROVIDER_VALUE CustomProviderTags = "Crossplane-compute-provider"
)

func (p CustomProviderTags) String() string {
	return string(p)
}
