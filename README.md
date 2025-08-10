# Custom Compute Provider for Crossplane - Study Project

> A learning project to understand Crossplane provider development for AWS EC2 instances.

## Overview

This project demonstrates how to create a custom provider for Crossplane, focusing on managing EC2 instances in AWS. It serves as an educational resource for understanding Crossplane provider development patterns and practices.

## Features

- EC2 instance provisioning
- Volume management
- Security group configuration
- Instance tagging
- Basic AWS networking setup

## Usage

### Provider Configuration

```yaml
apiVersion: compute.customcomputeprovider.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: compute-provider
spec:
  credentials:
    source: Secret
    secretRef:
      name: aws-secret
      namespace: crossplane-system
      key: credentials
```

### EC2 Instance Example

```yaml
apiVersion: compute.customcomputeprovider.crossplane.io/v1alpha1
kind: Compute
metadata:
  name: compute-cp
spec:
  forProvider:
    awsConfig:
      region: "us-east-1"
    instanceConfig:
      name: "cp-crossplane1"
      type: "t2.micro"
      ami: "ami-05b10e08d247fb927"

      storage:
        - deviceName: "/dev/xvda"
          diskType: "gp3"
          diskSize: 30

      networking:
        subnetID: "subnet-0f3031cfcab95eb28"
        securityGroups:
          - sg-0b3a670bdc8f7d07f

      tags:
        "Environment": "Dev"
        "Iac": "Crossplane"
        "Fe": "Backstage"
        "GitOps": "ArgoCD"

  providerConfigRef:
    name: compute-provider
```

## Prerequisites

- Kubernetes cluster
- Crossplane installed
- AWS credentials
- Basic understanding of Kubernetes and AWS

## Project Structure

The provider implements standard Crossplane interfaces:

- `Create`: Provisions new EC2 instances
- `Observe`: Monitors instance state
- `Update`: Modifies existing instances
- `Delete`: Terminates instances

## Learning Outcomes

Through this project, you can learn:

- Crossplane provider architecture
- AWS SDK integration
- Kubernetes custom resource definitions
- Cloud resource reconciliation patterns
- Infrastructure as Code principles

## Important Note

This is a study project intended for learning purposes. It should not be used in production environments.

## License

Licensed under the Apache License, Version 2.0
