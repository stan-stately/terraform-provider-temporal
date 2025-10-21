terraform {
  required_providers {
    temporal = {
      source  = "stately-stan/temporal"
      version = "0.1.2"
    }
  }
}

provider "temporal" {
  address   = "localhost:8233"
  namespace = "default"
  # api_key = var.temporal_api_key  # Uncomment for Temporal Cloud
  # tls     = true                  # Uncomment for Temporal Cloud
}
