resource "google_cloud_run_v2_service" "backend" {
  name                = local.service_name
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = true

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
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }

      env {
        name  = "GIN_MODE"
        value = "release"
      }

      volume_mounts {
        name       = "mlit-api-key"
        mount_path = "/secrets"
      }

      volume_mounts {
        name       = "app-internal-api-key"
        mount_path = "/secrets"
      }

      volume_mounts {
        name       = "google-maps-api-key"
        mount_path = "/secrets"
      }

      dynamic "env" {
        for_each = var.gemini_api_key != "" ? [1] : []
        content {
          name = "GEMINI_API_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.gemini_api_key.secret_id
              version = "latest"
            }
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

    volumes {
      name = "mlit-api-key"
      secret {
        secret = google_secret_manager_secret.mlit_api_key.secret_id
        items {
          path    = "mlit-api-key"
          version = "latest"
        }
      }
    }

    volumes {
      name = "app-internal-api-key"
      secret {
        secret = google_secret_manager_secret.app_internal_api_key.secret_id
        items {
          path    = "app-internal-api-key"
          version = "latest"
        }
      }
    }

    volumes {
      name = "google-maps-api-key"
      secret {
        secret = google_secret_manager_secret.google_maps_api_key.secret_id
        items {
          path    = "google-maps-api-key"
          version = "latest"
        }
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
    google_secret_manager_secret_iam_member.google_maps_accessor,
    google_secret_manager_secret_iam_member.gemini_accessor,
    google_project_iam_member.backend_trace_agent,
    google_project_iam_member.backend_metric_writer,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  name     = google_cloud_run_v2_service.backend.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "allUsers"
}
