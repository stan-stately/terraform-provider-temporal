terraform {
  required_providers {
    temporal = {
      source  = "stately-stan/temporal"
      version = "0.1.2"
    }
  }
}

provider "temporal" {
  address   = "us-west-2.aws.api.temporal.io:7233"
  namespace = "your-namespace"
  api_key   = var.temporal_api_key
  tls       = true
}

variable "temporal_api_key" {
  description = "Temporal Cloud API key"
  type        = string
  sensitive   = true
}
