# Terraform manages only the secret containers. Secret values (versions) would be
# stored in plaintext in tfstate, so they are added out-of-band with
# `gcloud secrets versions add` (see docs/security.md, #886).

resource "google_secret_manager_secret" "mlit_api_key" {
  secret_id = "mlit-api-key-${var.env}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "app_internal_api_key" {
  secret_id = "app-internal-api-key-${var.env}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "gemini_api_key" {
  secret_id = "gemini-api-key-${var.env}"

  replication {
    auto {}
  }
}

# Drop the previously managed versions from state without destroying them in GCP.
# Safe to delete these blocks once every environment has been applied.
removed {
  from = google_secret_manager_secret_version.mlit_api_key

  lifecycle {
    destroy = false
  }
}

removed {
  from = google_secret_manager_secret_version.app_internal_api_key

  lifecycle {
    destroy = false
  }
}

removed {
  from = google_secret_manager_secret_version.gemini_api_key

  lifecycle {
    destroy = false
  }
}
