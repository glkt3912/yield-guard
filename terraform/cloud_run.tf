resource "google_cloud_run_v2_service" "backend" {
  name                = local.service_name
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    service_account = google_service_account.backend.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      # Placeholder image for initial provisioning; GitHub Actions replaces this on first deploy.
      # lifecycle.ignore_changes prevents Terraform from reverting it afterwards.
      image = "us-docker.pkg.dev/cloudrun/container/hello:latest"

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle          = true
        startup_cpu_boost = true
      }

      ports {
        container_port = 8080
      }

      env {
        name  = "ALLOW_ORIGINS"
        value = var.vercel_frontend_url
      }

      env {
        name = "MLIT_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.mlit_api_key.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "APP_INTERNAL_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.app_internal_api_key.secret_id
            version = "latest"
          }
        }
      }

      liveness_probe {
        http_get {
          path = "/health"
        }
        initial_delay_seconds = 10
        period_seconds        = 30
        failure_threshold     = 3
      }

      startup_probe {
        http_get {
          path = "/health"
        }
        initial_delay_seconds = 5
        period_seconds        = 5
        failure_threshold     = 10
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
    ]
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.mlit_accessor,
    google_secret_manager_secret_iam_member.internal_key_accessor,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  name     = google_cloud_run_v2_service.backend.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "allUsers"
}
