package volume

import (
	"context"

	"github.com/crossplane/provider-customcomputeprovider/internal/provider"
)

type VolumeCommand interface {
	Run(ctx context.Context, c *provider.EC2Client) error
	GetType() string
}

type BaseCommand struct {
	commandType string
}

func (b *BaseCommand) GetType() string {
	return b.commandType
}
