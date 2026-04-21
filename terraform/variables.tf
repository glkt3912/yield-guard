variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Cloud Run and Artifact Registry"
  type        = string
  default     = "asia-northeast1"
}

variable "env" {
  description = "Deployment environment (e.g. prod, stg)"
  type        = string
}

variable "mlit_api_key" {
  description = "MLIT Reinfolib API subscription key"
  type        = string
  sensitive   = true
}

variable "app_internal_api_key" {
  description = "Shared secret for Vercel-to-Cloud-Run internal auth"
  type        = string
  sensitive   = true
}

variable "vercel_frontend_url" {
  description = "Vercel frontend URL set as ALLOW_ORIGINS on the backend"
  type        = string
}
