terraform {
  required_providers {
    temporal = {
      source  = "pgbi/temporal"
      version = "0.1.2"
    }
  }
}

provider "temporal" {
  address   = "your-namespace.tmprl.cloud:7233"
  namespace = "your-namespace"
  api_key   = var.temporal_api_key
  tls       = true
}

variable "temporal_api_key" {
  description = "Temporal Cloud API key"
  type        = string
  sensitive   = true
}
