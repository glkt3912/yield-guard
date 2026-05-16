# ─────────────────────────────────────────────────────
# 予算アラート (#256)
# 前提: Deployer SA に請求アカウントレベルの roles/billing.admin が必要。
# google_billing_account_iam_member.deployer_billing_admin で Terraform 管理済み。
# 初回ブートストラップのみ手動付与が必要:
#   gcloud billing accounts add-iam-policy-binding <BILLING_ACCOUNT_ID> \
#     --member="serviceAccount:<deployer-sa-email>" \
#     --role="roles/billing.admin"
# ─────────────────────────────────────────────────────
# Firestore 3機能（AIサマリーキャッシュ・ジオコードキャッシュ・ウォッチリスト）導入に伴う
# 月次コスト早期検知アラート: 1,000円 (80%/100%予測) で通知する。
resource "google_billing_budget" "firestore_early_alert" {
  provider        = google.billing
  billing_account = var.billing_account_id
  display_name    = "Yield Guard Firestore 早期コストアラート"

  budget_filter {
    projects = ["projects/${var.project_id}"]
    services = ["services/95FF-2EF5-5EA1"] # Cloud Firestore
  }

  amount {
    specified_amount {
      currency_code = "JPY"
      units         = "1000"
    }
  }

  threshold_rules {
    threshold_percent = 0.8
  }
  threshold_rules {
    threshold_percent = 1.0
    spend_basis       = "FORECASTED_SPEND"
  }

  all_updates_rule {
    monitoring_notification_channels = [
      google_monitoring_notification_channel.email.id,
    ]
    disable_default_iam_recipients = true
  }

  depends_on = [google_project_service.billingbudgets]

  lifecycle {
    ignore_changes = [
      amount,
      threshold_rules,
      all_updates_rule,
    ]
  }
}

resource "google_billing_budget" "monthly" {
  provider        = google.billing
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
    threshold_percent = 0.8
  }
  threshold_rules {
    threshold_percent = 1.0
  }

  all_updates_rule {
    monitoring_notification_channels = [
      google_monitoring_notification_channel.email.id,
    ]
    pubsub_topic                   = google_pubsub_topic.billing_alerts.id
    disable_default_iam_recipients = true
  }

  depends_on = [google_project_service.billingbudgets]

  lifecycle {
    # The Billing Budgets API rejects write operations from WIF-issued service account
    # tokens (403), even with billing.admin on the billing account. Human user credentials
    # (gcloud / GCP Console) succeed. This is a known limitation of WIF + personal billing
    # accounts. Budget values must be changed manually outside of CI.
    ignore_changes = [
      amount,
      threshold_rules,
      all_updates_rule,
    ]
  }
}
