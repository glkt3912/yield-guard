# ─────────────────────────────────────────────────────
# 通知チャンネル
# ─────────────────────────────────────────────────────
resource "google_monitoring_notification_channel" "email" {
  display_name = "Yield Guard アラート通知"
  type         = "email"
  labels = {
    email_address = var.notification_email
  }
}

# ─────────────────────────────────────────────────────
# ダッシュボード (#252)
# OTel カスタムメトリクス: workload.googleapis.com/<name>
# Cloud Run 標準メトリクス: run.googleapis.com/<name>
# ─────────────────────────────────────────────────────
resource "google_monitoring_dashboard" "yield_guard" {
  dashboard_json = jsonencode({
    displayName = "Yield Guard"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          xPos   = 0
          yPos   = 0
          width  = 6
          height = 4
          widget = {
            title = "MLIT API 応答時間 P99 (s)"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"workload.googleapis.com/mlit.api.request.duration\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_PERCENTILE_99"
                      crossSeriesReducer = "REDUCE_MEAN"
                      groupByFields      = []
                    }
                  }
                }
                plotType   = "LINE"
                targetAxis = "Y1"
              }]
              yAxis = { label = "秒", scale = "LINEAR" }
            }
          }
        },
        {
          xPos   = 6
          yPos   = 0
          width  = 6
          height = 4
          widget = {
            title = "キャッシュ ヒット / ミス (req/s)"
            xyChart = {
              dataSets = [
                {
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type=\"workload.googleapis.com/mlit.cache.hits\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                        groupByFields      = []
                      }
                    }
                  }
                  legendTemplate = "hits"
                  plotType       = "LINE"
                  targetAxis     = "Y1"
                },
                {
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type=\"workload.googleapis.com/mlit.cache.misses\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_RATE"
                        crossSeriesReducer = "REDUCE_SUM"
                        groupByFields      = []
                      }
                    }
                  }
                  legendTemplate = "misses"
                  plotType       = "LINE"
                  targetAxis     = "Y1"
                }
              ]
              yAxis = { label = "req/s", scale = "LINEAR" }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 4
          width  = 6
          height = 4
          widget = {
            title = "投資分析 API 呼び出し数 (req/s)"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"workload.googleapis.com/analyze.requests.total\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                      groupByFields      = []
                    }
                  }
                }
                plotType   = "STACKED_BAR"
                targetAxis = "Y1"
              }]
              yAxis = { label = "req/s", scale = "LINEAR" }
            }
          }
        },
        {
          xPos   = 6
          yPos   = 4
          width  = 6
          height = 4
          widget = {
            title = "Cloud Run 5xx エラー (req/s)"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"run.googleapis.com/request_count\" AND resource.labels.service_name=\"${local.service_name}\" AND metric.labels.response_code_class=\"5xx\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                      groupByFields      = []
                    }
                  }
                }
                plotType   = "LINE"
                targetAxis = "Y1"
              }]
              yAxis = { label = "req/s", scale = "LINEAR" }
            }
          }
        }
      ]
    }
  })
}

# ─────────────────────────────────────────────────────
# アラートポリシー (#253)
# ─────────────────────────────────────────────────────

# 1. MLIT API P99 応答時間 > 15s が 5 分継続
resource "google_monitoring_alert_policy" "mlit_latency" {
  display_name = "[Yield Guard] MLIT API P99 応答時間 > 15s"
  combiner     = "OR"

  conditions {
    display_name = "MLIT API P99 > 15s が 5 分継続"
    condition_monitoring_query_language {
      query    = <<-EOT
        fetch generic_task
        | metric 'workload.googleapis.com/mlit.api.request.duration'
        | align delta(5m)
        | every 5m
        | percentile(99)
        | condition val() > 15 "s"
      EOT
      duration = "300s"
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]
  alert_strategy {
    auto_close = "3600s"
  }
}

# 2. Cloud Run 5xx エラー率 > 5% が 5 分継続
# ratio = 5xx_count / total_count
resource "google_monitoring_alert_policy" "cloudrun_error_rate" {
  display_name = "[Yield Guard] Cloud Run 5xx エラー率 > 5%"
  combiner     = "OR"

  conditions {
    display_name = "5xx 割合 > 5% が 5 分継続"
    condition_monitoring_query_language {
      query    = <<-EOT
        fetch cloud_run_revision
        | metric 'run.googleapis.com/request_count'
        | filter resource.service_name = '${local.service_name}'
        | {
            filter metric.response_code_class = '5xx'
            | align rate(5m)
            | every 5m
          ;
            align rate(5m)
            | every 5m
          }
        | ratio
        | condition val() > 0.05
      EOT
      duration = "300s"
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]
  alert_strategy {
    auto_close = "3600s"
  }
}

# 3. キャッシュヒット率 < 50% が 10 分継続
# ratio = misses / (misses + hits) → alert when > 0.5 (= hit rate < 50%)
resource "google_monitoring_alert_policy" "cache_hit_rate" {
  display_name = "[Yield Guard] キャッシュヒット率 < 50%"
  combiner     = "OR"

  conditions {
    display_name = "キャッシュヒット率 < 50% が 10 分継続"
    # fetch generic_task: OTel exporter が GCP リソース検出器なしの場合のデフォルト。
    # setup.go で resource.WithDetectors() を使っていないため generic_task が正しいとコード解析で確認 (#269)。
    # apply 後は Metrics Explorer で workload.googleapis.com/mlit.cache.misses のリソースタイプを目視確認すること。
    condition_monitoring_query_language {
      query = <<-EOT
        fetch generic_task
        | {
            metric 'workload.googleapis.com/mlit.cache.misses'
            | align rate(10m)
            | every 10m
          ;
            metric 'workload.googleapis.com/mlit.cache.hits'
            | align rate(10m)
            | every 10m
          }
        | ratio
        | condition val() > 0.5
      EOT
      # 30分継続で判定: 再起動直後のキャッシュリセット（約10分）による誤検知を抑制 (#270)
      # notification_rate_limit はログベースアラート専用のため MQL アラートには使用不可
      duration = "1800s"
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]
  alert_strategy {
    auto_close = "3600s"
  }
}

# 4. Cloud Scheduler ウォームアップジョブが連続失敗
resource "google_monitoring_alert_policy" "warmup_job_failure" {
  display_name = "[Yield Guard] ウォームアップジョブが連続失敗"
  combiner     = "OR"

  conditions {
    display_name = "warmup-cache または warmup-ping が 30 分以内に 2 回以上失敗"
    condition_threshold {
      filter = join(" AND ", [
        "metric.type=\"cloudscheduler.googleapis.com/job/attempt_count\"",
        "resource.type=\"cloud_scheduler_job\"",
        "metric.labels.status=\"FAILED\"",
        "resource.labels.job_id=monitoring.regex.full_match(\"warmup-.*-${var.env}\")",
      ])
      comparison      = "COMPARISON_GT"
      threshold_value = 1
      duration        = "0s"
      aggregations {
        alignment_period     = "1800s"
        per_series_aligner   = "ALIGN_COUNT_TRUE"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.labels.job_id"]
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]
  alert_strategy {
    auto_close = "86400s"
  }
}

# 5. Cloud Run インスタンス数が上限 (max_instance_count=2) に到達し 5 分継続
resource "google_monitoring_alert_policy" "cloudrun_max_instances" {
  display_name = "[Yield Guard] Cloud Run インスタンス数が上限に到達"
  combiner     = "OR"

  conditions {
    display_name = "インスタンス数 > 1 が 5 分継続（上限 2 に到達）"
    condition_threshold {
      filter          = "metric.type=\"run.googleapis.com/container/instance_count\" AND resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${local.service_name}\""
      comparison      = "COMPARISON_GT"
      threshold_value = 1
      duration        = "300s"
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MAX"
        cross_series_reducer = "REDUCE_MAX"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]
  alert_strategy {
    auto_close = "3600s"
  }
}
