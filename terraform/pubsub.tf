resource "google_pubsub_topic" "billing_alerts" {
  name    = "billing-alerts-${var.env}"
  project = var.project_id

  depends_on = [google_project_service.apis]
}

