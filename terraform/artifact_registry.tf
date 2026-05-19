# Artifact Registry は prod 専用。stg は prod の同一リポジトリを共用する。
# repository_id に env suffix がないため stg で作ろうとすると既存リソースと競合する。
resource "google_artifact_registry_repository" "backend" {
  count         = var.env == "prod" ? 1 : 0
  repository_id = "yield-guard"
  location      = var.region
  format        = "DOCKER"
  description   = "yield-guard backend Docker images"

  # Ensure the deployer IAM binding is applied before updating repository metadata.
  # Without this, Terraform applies IAM and repository changes in parallel, causing
  # a 403 on repositories.update before the new IAM role has propagated.
  depends_on = [google_project_iam_member.deployer_ar_repo_admin]

  # false にしないとドライラン扱いになり実際に削除されない。
  cleanup_policy_dry_run = false

  # KEEP が DELETE より優先される。最新5件は保護され、それ以外は削除対象になる。
  cleanup_policies {
    id     = "keep-5-most-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }

  # older_than を長く設定すると「6枚目以降かつ期間内」のイメージが残り続ける。
  # プッシュ直後の競合を避けるため 1 日のバッファのみ設ける。
  cleanup_policies {
    id     = "delete-old-tagged"
    action = "DELETE"
    condition {
      tag_state  = "TAGGED"
      older_than = "86400s" # 1日
    }
  }

  cleanup_policies {
    id     = "delete-untagged"
    action = "DELETE"
    condition {
      tag_state  = "UNTAGGED"
      older_than = "86400s" # 1日
    }
  }
}
