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

resource "google_project_iam_member" "deployer_sa_admin" {
  # Required to set IAM policies on individual service accounts (WIF binding, act-as).
  project = var.project_id
  role    = "roles/iam.serviceAccountAdmin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_ar_repo_admin" {
  count = var.env == "prod" ? 1 : 0
  # artifactregistry.admin is required to update repository metadata (cleanup_policies etc.)
  # via terraform apply. repoAdmin does not include artifactregistry.repositories.update.
  # stg does not manage the AR repository (prod-only); writer is already granted via deployer_push.
  project = var.project_id
  role    = "roles/artifactregistry.admin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_artifact_registry_repository_iam_member" "deployer_push" {
  # リポジトリは prod TF が作成するが、stg deployer も同リポジトリにプッシュする。
  # google_artifact_registry_repository.backend は prod 専用 (count=0 for stg) なので
  # リソース参照ではなくリポジトリ ID を直接指定する。
  # depends_on でリポジトリ作成後に IAM を付与することを保証する（prod 初回 apply 時の競合防止）。
  # stg では backend が count=0 のため depends_on は空依存となり副作用なし。
  depends_on = [google_artifact_registry_repository.backend]
  repository = "yield-guard"
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

resource "google_service_account_iam_member" "deployer_act_as_billing_stop" {
  count = var.env == "prod" ? 1 : 0
  # deployer SA impersonates billing_stop SA when deploying Cloud Function.
  service_account_id = google_service_account.billing_stop[0].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_service_account_iam_member" "deployer_act_as_default_compute" {
  count = var.env == "prod" ? 1 : 0
  # Cloud Functions Gen2 build (Cloud Build) runs as the default Compute SA.
  # deployer SA must be able to act as it to submit the build job.
  service_account_id = "projects/${var.project_id}/serviceAccounts/${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_storage_bucket_iam_member" "deployer_tfstate_admin" {
  # storage.admin includes getIamPolicy/setIamPolicy needed for terraform apply.
  bucket = "yield-guard-tfstate"
  role   = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_monitoring_editor" {
  count = var.env == "prod" ? 1 : 0
  # Required to manage Cloud Monitoring dashboards, alert policies, and notification channels via terraform apply.
  # monitoring.editor covers CRUD for dashboards/alerts/channels without IAM management (unlike monitoring.admin).
  # stg has no monitoring resources (all guarded with count = prod ? 1 : 0).
  project = var.project_id
  role    = "roles/monitoring.editor"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_secret_manager_admin" {
  # Required to create and manage Secret Manager secrets via terraform apply.
  project = var.project_id
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_storage_admin" {
  count = var.env == "prod" ? 1 : 0
  # Required to create GCS buckets (e.g. Cloud Function source bucket) via terraform apply.
  # stg has no GCS buckets to manage. tfstate access is granted separately via deployer_tfstate_admin.
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_pubsub_editor" {
  # Required to create Pub/Sub topics and subscriptions via terraform apply.
  # pubsub.editor covers create/delete/update without IAM management (unlike pubsub.admin).
  project = var.project_id
  role    = "roles/pubsub.editor"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_cloudfunctions_admin" {
  count = var.env == "prod" ? 1 : 0
  # Required to deploy Cloud Functions Gen2 via terraform apply.
  # stg has no Cloud Functions.
  project = var.project_id
  role    = "roles/cloudfunctions.admin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_logging_config_writer" {
  count = var.env == "prod" ? 1 : 0
  # Required to create and manage log-based metrics via terraform apply.
  # stg has no log-based metrics (warmup_job_failure metric is prod-only).
  project = var.project_id
  role    = "roles/logging.configWriter"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_firestore_owner" {
  count = var.env == "prod" ? 1 : 0
  # Required to create Firestore databases and TTL policies via terraform apply.
  # stg does not manage Firestore; it reuses the prod (default) database.
  project = var.project_id
  role    = "roles/datastore.owner"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_cloudscheduler_admin" {
  # Required to create and manage Cloud Scheduler jobs via terraform apply.
  project = var.project_id
  role    = "roles/cloudscheduler.admin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_billing_account_iam_member" "deployer_billing_admin" {
  count    = var.env == "prod" ? 1 : 0
  provider = google.billing
  # Required to create and update Billing Budgets via terraform apply.
  # billing.admin is needed; billing.costsManager does not include budgets.update in practice.
  billing_account_id = var.billing_account_id
  role               = "roles/billing.admin"
  member             = "serviceAccount:${google_service_account.deployer.email}"

  depends_on = [google_project_service.cloudbilling]
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

resource "google_secret_manager_secret_iam_member" "gemini_accessor" {
  secret_id = google_secret_manager_secret.gemini_api_key.secret_id
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

resource "google_project_iam_member" "backend_firestore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.backend.email}"
}

# ─── Billing Stop Cloud Function SA (prod 専用) ────────────────────────────────
resource "google_service_account" "billing_stop" {
  count        = var.env == "prod" ? 1 : 0
  account_id   = "sa-billing-stop-${var.env}"
  display_name = "yield-guard ${var.env} billing stop (Cloud Function)"
}

resource "google_project_iam_member" "billing_stop_run_developer" {
  count   = var.env == "prod" ? 1 : 0
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.billing_stop[0].email}"
}

resource "google_storage_bucket_iam_member" "billing_stop_storage_viewer" {
  count  = var.env == "prod" ? 1 : 0
  bucket = google_storage_bucket.functions_source[0].name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.billing_stop[0].email}"
}
