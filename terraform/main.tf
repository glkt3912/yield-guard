terraform {
  required_version = ">= 1.14"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # GCS remote backend でシークレットを含む state をローカル管理から保護する。
  # bucket は手動作成（terraform.tfvars.example 参照）してからコメントを外す。
  # backend "gcs" {
  #   bucket = "<project_id>-tfstate"
  #   prefix = "yield-guard/prod"
  # }
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
