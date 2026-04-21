terraform {
  required_version = ">= 1.9"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

locals {
  service_name = "yield-guard-${var.env}-backend"
  sa_name      = "sa-yield-guard-${var.env}"
  sa_email     = "${local.sa_name}@${var.project_id}.iam.gserviceaccount.com"
  image_repo   = "${var.region}-docker.pkg.dev/${var.project_id}/yield-guard/backend"
}
