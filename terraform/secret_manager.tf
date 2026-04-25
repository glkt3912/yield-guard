resource "google_secret_manager_secret" "mlit_api_key" {
  secret_id = "mlit-api-key-${var.env}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "mlit_api_key" {
  secret      = google_secret_manager_secret.mlit_api_key.id
  secret_data = var.mlit_api_key
}

resource "google_secret_manager_secret" "app_internal_api_key" {
  secret_id = "app-internal-api-key-${var.env}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "app_internal_api_key" {
  secret      = google_secret_manager_secret.app_internal_api_key.id
  secret_data = var.app_internal_api_key
}

resource "google_secret_manager_secret" "google_maps_api_key" {
  secret_id = "google-maps-api-key-${var.env}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "google_maps_api_key" {
  secret      = google_secret_manager_secret.google_maps_api_key.id
  secret_data = var.google_maps_api_key
}
