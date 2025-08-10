package updater

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type Updater interface {
	Execute(ctx UpdateContext) error
	GetType() string
}

type UpdateContext struct {
	Context context.Context
	Current *types.Instance
	Desired *v1alpha1.InstanceConfig
	Client  *provider.EC2Client
	Logger  logging.Logger
}

type BaseOperation struct {
	opType string
	logger logging.Logger
}

func (b *BaseOperation) GetType() string {
	return b.opType
}

type UpdateOrchestrator struct {
	operations map[string]Updater
	logger     logging.Logger
}

func NewUpdateOrchestrator(logger logging.Logger) *UpdateOrchestrator {
	ops := make(map[string]Updater)
	ops["TAG"] = NewTagOperation(logger)
	ops["NAME"] = NewTagOperation(logger)
	ops["VOLUME"] = NewVolumeOperation(logger)
	ops["TYPE"] = NewTypeUpdateOperation(logger)
	ops["SECURITY_GROUP"] = NewSecurityGroupUpdateOperation(logger)

	return &UpdateOrchestrator{
		operations: ops,
		logger:     logger,
	}
}

func (o *UpdateOrchestrator) ExecuteUpdates(updateContext UpdateContext, updates map[string]bool) error {
	for opType, needsUpdate := range updates {
		if !needsUpdate {
			continue
		}

		op, exists := o.operations[opType]
		if !exists {
			o.logger.Info("no operation registered for type", "type", opType)
			continue
		}

		o.logger.Info("executing update operation", "type", opType)
		if err := op.Execute(updateContext); err != nil {
			return err
		}

		o.logger.Info("successfully completed update operation", "type", opType)
	}
	return nil
}
