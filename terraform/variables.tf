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

variable "gemini_api_key" {
  description = "Google AI Studio API key for Gemini AI investment summary"
  type        = string
  sensitive   = true
  default     = ""
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
