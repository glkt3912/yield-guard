resource "google_cloud_scheduler_job" "warmup_ping" {
  name             = "warmup-ping-${var.env}"
  region           = var.region
  schedule         = "*/15 * * * *"
  time_zone        = "Asia/Tokyo"
  attempt_deadline = "30s"

  http_target {
    uri         = "${google_cloud_run_v2_service.backend.uri}/health"
    http_method = "GET"
  }
}

resource "google_cloud_scheduler_job" "warmup_cache" {
  name             = "warmup-cache-${var.env}"
  region           = var.region
  schedule         = "0 4 * * *"
  time_zone        = "Asia/Tokyo"
  attempt_deadline = "60s"

  http_target {
    uri         = "${google_cloud_run_v2_service.backend.uri}/warm"
    http_method = "POST"
    headers = {
      "Content-Type"   = "application/json"
      "X-Internal-Key" = var.app_internal_api_key
    }
    body = base64encode("{}")
  }
}
