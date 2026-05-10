resource "google_pubsub_topic" "billing_alerts" {
  name    = "billing-alerts-${var.env}"
  project = var.project_id

  depends_on = [
    google_project_service.apis,
    google_project_iam_member.deployer_pubsub_editor,
  ]
}

