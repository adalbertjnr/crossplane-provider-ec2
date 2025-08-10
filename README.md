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
- Drift detection - For **name**, **instance type**, **storage**, **security groups** and **tags** (currently change subnets or ami is a todo proccess still)

## Prerequisites

- Kubernetes cluster
- Crossplane installed
- AWS credentials
- Basic understanding of Kubernetes and AWS

## Usage

### CRDs

The CRDs under package/crds must be installed first

> kubectl apply -f package/crds

### Provider Configuration

This is necessary for autentication outside aws ec2/nodes
Just provide a secret containing the aws secret and access key with permissions to manage ec2

Secret and providerConfig are not necessary if the provider is installed inside any ec2 or eks node, but they need to have policies attached to an instance profile.

### Secret

```yaml
apiVersion: v1
data:
  credentials: {"access_key_id":"" secret_access_key":""}
kind: Secret
metadata:
  name: compute-secret
  namespace: crossplane-system
type: Opaque
```

### ProviderConfig

```yaml
apiVersion: compute.customcomputeprovider.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: compute-provider
spec:
  credentials:
    source: Secret
    secretRef:
      name: compute-secret
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
