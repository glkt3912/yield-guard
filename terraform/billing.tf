# ─────────────────────────────────────────────────────
# 予算アラート (#256)
# 前提: Deployer SA に請求アカウントレベルの roles/billing.costsManager が必要。
# Terraform 管理外のため初回のみ手動付与:
#   gcloud billing accounts add-iam-policy-binding <BILLING_ACCOUNT_ID> \
#     --member="serviceAccount:<deployer-sa-email>" \
#     --role="roles/billing.costsManager"
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
      units         = "10000"
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
