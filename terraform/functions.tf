# Cloud Function のソースコードをバケットにアップロード。
# バケット名はグローバルユニークのため prod 専用。stg は prod バケットを共用しない
#（stg に billing stop Function 自体が不要なため）。
resource "google_storage_bucket" "functions_source" {
  count                       = var.env == "prod" ? 1 : 0
  name                        = "${var.project_id}-functions-source"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true

  depends_on = [google_project_iam_member.deployer_storage_admin]
}

data "archive_file" "billing_stop" {
  count       = var.env == "prod" ? 1 : 0
  type        = "zip"
  source_dir  = "${path.module}/functions/billing_stop"
  output_path = "${path.module}/functions/billing_stop.zip"
}

resource "google_storage_bucket_object" "billing_stop" {
  count  = var.env == "prod" ? 1 : 0
  name   = "billing_stop_${data.archive_file.billing_stop[0].output_md5}.zip"
  bucket = google_storage_bucket.functions_source[0].name
  source = data.archive_file.billing_stop[0].output_path
}

resource "google_cloudfunctions2_function" "billing_stop" {
  count    = var.env == "prod" ? 1 : 0
  name     = "billing-stop-${var.env}"
  location = var.region
  project  = var.project_id

  build_config {
    runtime     = "python312"
    entry_point = "stop_cloud_run"
    source {
      storage_source {
        bucket = google_storage_bucket.functions_source[0].name
        object = google_storage_bucket_object.billing_stop[0].name
      }
    }
  }

  service_config {
    service_account_email = google_service_account.billing_stop.email
    environment_variables = {
      PROJECT_ID   = var.project_id
      REGION       = var.region
      SERVICE_NAME = local.service_name
    }
    max_instance_count = 1
    available_memory   = "128Mi"
    timeout_seconds    = 60
  }

  event_trigger {
    trigger_region = var.region
    event_type     = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic   = google_pubsub_topic.billing_alerts.id
    retry_policy   = "RETRY_POLICY_RETRY"
  }

  depends_on = [google_project_service.apis]
}
