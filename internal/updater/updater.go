package updater

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/provider-customcomputeprovider/apis/compute/v1alpha1"
	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
	ot "github.com/crossplane/provider-customcomputeprovider/internal/types"
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
	ops[ot.NAME.String()] = NewNameOperation(logger)
	ops[ot.SECURITY_GROUPS.String()] = NewSecurityGroupUpdateOperation(logger)
	ops[ot.TAGS.String()] = NewTagOperation(logger)
	ops[ot.INSTANCE_TYPE.String()] = NewTypeUpdateOperation(logger)
	ops[ot.VOLUME.String()] = NewVolumeOperation(logger)

	return &UpdateOrchestrator{
		operations: ops,
		logger:     logger,
	}
}

func (o *UpdateOrchestrator) ExecuteUpdates(updateContext UpdateContext, updates map[string]bool) error {
	order := []string{
		ot.NAME.String(),
		ot.TAGS.String(),
		ot.SECURITY_GROUPS.String(),
		ot.INSTANCE_TYPE.String(),
		ot.VOLUME.String(),
	}
	o.logger.Info("starting updates execution",
		"updates_needed", updates,
		"execution_order", order)

	for _, opType := range order {
		needsUpdate, exists := updates[opType]
		if !exists || !needsUpdate {
			o.logger.Info("skipping operation", "type", opType)
			continue
		}

		op, exists := o.operations[opType]
		if !exists {
			o.logger.Info("no operation registered for type", "type", opType)
			continue
		}

		o.logger.Info("executing update operation",
			"type", opType,
			"current_state", map[string]interface{}{
				"instance_id": *updateContext.Current.InstanceId,
				"state":       updateContext.Current.State.Name,
			})

		if err := op.Execute(updateContext); err != nil {
			o.logger.Info("failed to execute operation",
				"type", opType,
				"error", err)
			return err
		}

		if err := o.refreshInstanceState(&updateContext); err != nil {
			return err
		}

		o.logger.Info("successfully completed update operation",
			"type", opType,
			"new_state", updateContext.Current.State.Name)
	}
	return nil
}

func (o *UpdateOrchestrator) refreshInstanceState(ctx *UpdateContext) error {
	instance, err := ctx.Client.GetInstanceByID(ctx.Context, *ctx.Current.InstanceId)
	if err != nil {
		return err
	}
	ctx.Current = instance
	return nil
}
