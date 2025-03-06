package cloud

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func AWSClientConnector(ctx context.Context) func(region string) (aws.Config, error) {
	return func(region string) (aws.Config, error) {
		cfg, err := config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
		)
		if err != nil {
			return aws.Config{}, err
		}
		return cfg, nil
	}
}

type EC2Client struct {
	Client *ec2.Client
}

func NewEC2Client(c aws.Config) *EC2Client {
	client := ec2.NewFromConfig(c)

	return &EC2Client{Client: client}
}
