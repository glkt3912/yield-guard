output "cloud_run_url" {
  description = "Public URL of the Cloud Run service. For stg: set as BACKEND_URL in Vercel project settings → Environment Variables → Preview."
  value       = google_cloud_run_v2_service.backend.uri
}

output "wif_provider" {
  description = "Workload Identity Federation provider resource name (set as WIF_PROVIDER GitHub Secret)"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "deployer_sa_email" {
  description = "Deployer SA email (set as SA_EMAIL GitHub Secret)"
  value       = google_service_account.deployer.email
}

output "runtime_sa_email" {
  description = "Runtime SA email (used by Cloud Run)"
  value       = google_service_account.backend.email
}
