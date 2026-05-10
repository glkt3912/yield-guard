terraform {
  required_version = ">= 1.14"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.0"
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

# Billing Account-scoped resources (e.g. google_billing_budget) require a quota
# project to be specified explicitly; without it the API returns 403.
provider "google" {
  alias                 = "billing"
  project               = var.project_id
  region                = var.region
  user_project_override = true
  billing_project       = var.project_id
}

data "google_project" "project" {}

locals {
  service_name     = "yield-guard-${var.env}-backend"
  sa_name          = "sa-yield-guard-${var.env}"
  deployer_sa_name = "sa-yield-guard-${var.env}-deployer"
}
