locals {
  required_apis = toset([
    "run.googleapis.com",
    "artifactregistry.googleapis.com",
    "secretmanager.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "serviceusage.googleapis.com",
    "cloudtrace.googleapis.com",
    "monitoring.googleapis.com",
    "geocoding-backend.googleapis.com",
    # Managed explicitly to prevent unintended disablement after AR Standard scan deprecation (2025-07-31).
    # Vulnerability scanning is handled by Trivy in CI only; AR Advanced scanning is intentionally omitted.
    "containeranalysis.googleapis.com",
    "pubsub.googleapis.com",
    "cloudfunctions.googleapis.com",
    "cloudbuild.googleapis.com",
    # Required for Cloud Functions Gen2 Pub/Sub event triggers (Eventarc backend).
    "eventarc.googleapis.com",
  ])
}

resource "google_project_service" "apis" {
  for_each = local.required_apis
  service  = each.value

  # API を無効化しないことで他リソースへの影響を防ぐ
  disable_on_destroy = false
}

resource "google_project_service" "billingbudgets" {
  service            = "billingbudgets.googleapis.com"
  disable_on_destroy = false
}
