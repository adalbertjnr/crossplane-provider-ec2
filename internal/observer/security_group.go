package validation

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/provider-customcomputeprovider/pkg/generic"
)

type SecurityGroupValidator struct{}

func (v *SecurityGroupValidator) NeedsUpdate(ctx ValidationContext) bool {
	currentSecurityGroupIDs := ctx.Current.SecurityGroups
	desiredSecurityGroupIDs := ctx.Desired.Networking.InstanceSecurityGroups

	currentSGExtractorFunc := func(security types.GroupIdentifier) string { return *security.GroupId }
	desiredSGExtractorFunc := func(securityGroupId string) string { return securityGroupId }

	currentMapSGIds := generic.FromSliceToMap(
		currentSecurityGroupIDs,
		currentSGExtractorFunc,
	)

	desiredMapSGIds := generic.FromSliceToMap(
		desiredSecurityGroupIDs,
		desiredSGExtractorFunc,
	)

	for _, dsg := range desiredSecurityGroupIDs {
		if _, exists := currentMapSGIds[dsg]; !exists {
			return true
		}
	}

	for _, csg := range currentSecurityGroupIDs {
		if _, exists := desiredMapSGIds[*csg.GroupId]; !exists {
			return true
		}
	}

	return false
}

func (*SecurityGroupValidator) GetValidationType() string {
	return "SecurityGroup"
}
