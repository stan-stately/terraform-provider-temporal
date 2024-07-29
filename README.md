# Temporal Terraform Provider

## Overview

The Temporal Terraform Provider allows you to manage [Temporal](https://temporal.io/) resources using [Terraform](https://www.terraform.io/).

## Installation

To install the Temporal provider, add it to your Terraform configuration file as follows:

```hcl
terraform {
  required_providers {
    temporal = {
      source = "pgbi/temporal"
      version = "0.1.0"
    }
  }
}

provider "temporal" {
  address   = "localhost:7233"
  namespace = "default"
}
```