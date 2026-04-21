terraform {
  required_version = ">= 1.14"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  backend "gcs" {
    bucket = "yield-guard-tfstate"
    prefix = "yield-guard/prod"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

locals {
  service_name = "yield-guard-${var.env}-backend"
  sa_name      = "sa-yield-guard-${var.env}"
  image_repo   = "${var.region}-docker.pkg.dev/${var.project_id}/yield-guard/backend"
}
