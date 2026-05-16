resource "google_firestore_database" "default" {
  project     = var.project_id
  name        = "(default)"
  location_id = "asia-northeast1"
  type        = "FIRESTORE_NATIVE"
  depends_on  = [google_project_service.apis]
}

resource "google_firestore_field" "mlit_cache_ttl" {
  project    = var.project_id
  database   = google_firestore_database.default.name
  collection = "mlit_cache"
  field      = "expiresAt"
  ttl_config {}
}

resource "google_firestore_field" "geocode_cache_ttl" {
  project    = var.project_id
  database   = google_firestore_database.default.name
  collection = "geocode_cache"
  field      = "expiresAt"
  ttl_config {}
}
