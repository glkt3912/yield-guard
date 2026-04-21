resource "google_artifact_registry_repository" "backend" {
  repository_id = "yield-guard"
  location      = var.region
  format        = "DOCKER"
  description   = "yield-guard backend Docker images"

  cleanup_policies {
    id     = "keep-5-most-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }

  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"
    condition {
      tag_state = "TAGGED"
    }
  }
}
