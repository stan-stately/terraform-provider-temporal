terraform {
  required_providers {
    temporal = {
      source  = "pgbi/temporal"
      version = "0.1.1"
    }
  }
}

provider "temporal" {
  address   = "localhost:8233"
  namespace = "default"
}
