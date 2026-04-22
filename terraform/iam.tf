resource "google_service_account" "backend" {
  account_id   = local.sa_name
  display_name = "yield-guard ${var.env} backend"
}

# --- Workload Identity Federation ---

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-pool-${var.env}"
  display_name              = "GitHub Actions Pool (${var.env})"
  description               = "OIDC pool for GitHub Actions deployments"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "GitHub OIDC provider"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.repository" = "assertion.repository"
  }

  # Restrict token exchange to this specific repository
  attribute_condition = "attribute.repository == 'glkt3912/yield-guard'"
}

resource "google_service_account_iam_binding" "wif_impersonation" {
  service_account_id = google_service_account.backend.name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/glkt3912/yield-guard"
  ]
}

# --- Minimal permissions for the Service Account ---

resource "google_artifact_registry_repository_iam_member" "sa_push" {
  repository = google_artifact_registry_repository.backend.name
  location   = var.region
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.backend.email}"
}

resource "google_project_iam_member" "sa_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.backend.email}"
}

resource "google_service_account_iam_member" "sa_act_as" {
  # deploy-cloudrun Action が SA impersonation に必要。
  # GitHub Actions SA と Cloud Run ランタイム SA を統一しているための自己参照。
  # SA を分離する場合は GitHub Actions 側 SA → Cloud Run SA への付与に変更する。
  service_account_id = google_service_account.backend.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.backend.email}"
}

resource "google_secret_manager_secret_iam_member" "mlit_accessor" {
  secret_id = google_secret_manager_secret.mlit_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.backend.email}"
}

resource "google_secret_manager_secret_iam_member" "internal_key_accessor" {
  secret_id = google_secret_manager_secret.app_internal_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.backend.email}"
}

# --- Terraform state backend permissions ---

resource "google_storage_bucket_iam_member" "sa_tfstate_admin" {
  # terraform plan/apply writes a lock file; objectAdmin is required.
  bucket = "yield-guard-tfstate"
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.backend.email}"
}
