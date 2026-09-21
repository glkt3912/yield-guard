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

variable "vercel_frontend_url" {
  description = "Vercel frontend URL set as ALLOW_ORIGINS on the backend"
  type        = string
}

variable "enable_gemini_summary" {
  description = "Inject GEMINI_API_KEY into Cloud Run. Add a version to gemini-api-key-<env> via gcloud before enabling."
  type        = bool
  default     = false
}

variable "billing_account_id" {
  description = "GCP billing account ID for budget alerts (format: XXXXXX-XXXXXX-XXXXXX)"
  type        = string
}

variable "notification_email" {
  description = "Email address for Cloud Monitoring alert notifications"
  type        = string
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$", var.notification_email))
    error_message = "notification_email must be a valid email address."
  }
}
