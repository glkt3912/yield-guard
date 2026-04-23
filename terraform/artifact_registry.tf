resource "google_artifact_registry_repository" "backend" {
  repository_id = "yield-guard"
  location      = var.region
  format        = "DOCKER"
  description   = "yield-guard backend Docker images"

  # 評価順序: KEEP が優先され、最新5件は保護される。
  # それ以外のタグ付きイメージは older_than を超えた時点で DELETE 対象となる。
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
      tag_state  = "TAGGED"
      older_than = "2592000s" # 30日
    }
  }
}
