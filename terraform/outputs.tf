output "cloud_run_url" {
  description = "Public URL of the Cloud Run service"
  value       = google_cloud_run_v2_service.backend.uri
}

output "wif_provider" {
  description = "Workload Identity Federation provider resource name (set as WIF_PROVIDER GitHub Secret)"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "sa_email" {
  description = "Service account email (set as SA_EMAIL GitHub Secret)"
  value       = google_service_account.backend.email
}
