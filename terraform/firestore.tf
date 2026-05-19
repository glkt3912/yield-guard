# Firestore の (default) データベースはプロジェクト単一リソースのため prod 専用。
# stg 環境は既存の prod (default) DB を共用する（コレクション名でデータが分離される）。
resource "google_firestore_database" "default" {
  count       = var.env == "prod" ? 1 : 0
  project     = var.project_id
  name        = "(default)"
  location_id = "asia-northeast1"
  type        = "FIRESTORE_NATIVE"
  depends_on  = [google_project_service.apis]
}

resource "google_firestore_field" "mlit_cache_ttl" {
  count      = var.env == "prod" ? 1 : 0
  project    = var.project_id
  database   = google_firestore_database.default[0].name
  collection = "mlit_cache"
  field      = "expiresAt"
  ttl_config {}
}

resource "google_firestore_field" "geocode_cache_ttl" {
  count      = var.env == "prod" ? 1 : 0
  project    = var.project_id
  database   = google_firestore_database.default[0].name
  collection = "geocode_cache"
  field      = "expiresAt"
  ttl_config {}
}
