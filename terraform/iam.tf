# --- Service Accounts ---

resource "google_service_account" "backend" {
  account_id   = local.sa_name
  display_name = "yield-guard ${var.env} backend (runtime)"
}

resource "google_service_account" "deployer" {
  account_id   = local.deployer_sa_name
  display_name = "yield-guard ${var.env} deployer (CI/CD)"
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

  attribute_condition = "attribute.repository == 'glkt3912/yield-guard'"
}

resource "google_service_account_iam_binding" "wif_impersonation" {
  service_account_id = google_service_account.deployer.name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/glkt3912/yield-guard"
  ]
}

# --- Deployer SA permissions (CI/CD only) ---

resource "google_project_iam_member" "deployer_viewer" {
  project = var.project_id
  role    = "roles/viewer"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_project_iam_admin" {
  project = var.project_id
  role    = "roles/resourcemanager.projectIamAdmin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_artifact_registry_repository_iam_member" "deployer_push" {
  repository = google_artifact_registry_repository.backend.name
  location   = var.region
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_service_account_iam_member" "deployer_act_as_backend" {
  # deployer SA impersonates runtime SA when deploying Cloud Run.
  service_account_id = google_service_account.backend.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_storage_bucket_iam_member" "deployer_tfstate_admin" {
  # storage.admin includes getIamPolicy/setIamPolicy needed for terraform apply.
  bucket = "yield-guard-tfstate"
  role   = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.deployer.email}"
}

# --- Runtime SA permissions (Cloud Run only) ---

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

resource "google_project_iam_member" "backend_trace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.backend.email}"
}

resource "google_project_iam_member" "backend_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.backend.email}"
}
