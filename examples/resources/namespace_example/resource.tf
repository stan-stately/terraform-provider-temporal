resource "temporal_namespace" "example" {
  name          = "example"
  retention_ttl = "365d"
}
