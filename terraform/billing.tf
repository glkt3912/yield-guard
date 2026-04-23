# ─────────────────────────────────────────────────────
# 予算アラート (#256)
# ─────────────────────────────────────────────────────
resource "google_billing_budget" "monthly" {
  billing_account = var.billing_account_id
  display_name    = "Yield Guard 月次予算アラート"

  budget_filter {
    projects = ["projects/${var.project_id}"]
  }

  amount {
    specified_amount {
      currency_code = "JPY"
      units         = "1000"
    }
  }

  threshold_rules {
    threshold_percent = 0.5
  }
  threshold_rules {
    threshold_percent = 1.0
  }

  all_updates_rule {
    monitoring_notification_channels = [
      google_monitoring_notification_channel.email.id,
    ]
    disable_default_iam_recipients = true
  }

  depends_on = [google_project_service.billingbudgets]
}
