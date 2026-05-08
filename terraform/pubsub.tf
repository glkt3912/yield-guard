resource "google_pubsub_topic" "billing_alerts" {
  name    = "billing-alerts-${var.env}"
  project = var.project_id

  depends_on = [google_project_service.apis]
}

resource "google_pubsub_subscription" "billing_alerts" {
  name    = "billing-alerts-sub-${var.env}"
  topic   = google_pubsub_topic.billing_alerts.id
  project = var.project_id

  ack_deadline_seconds = 60

  expiration_policy {
    ttl = ""
  }
}
