
# Creates a CloudWatch Log Group for the application.
resource "aws_cloudwatch_log_group" "main" {
  name = "/ecs/${var.app_name}"

  tags = {
    Name = "${var.app_name}-logs"
  }
}

resource "aws_cloudwatch_log_group" "app_logs" {
  name = "/ecs/${var.app_name}-app"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "app_debug_logs" {
  name = "/ecs/debug-logs/${var.app_name}-app"
  retention_in_days = 7
}

resource "aws_cloudwatch_log_group" "adot_debug_logs" {
  name = "/ecs/debug-logs/adot-collector"
  retention_in_days = 7
}

resource "aws_cloudwatch_metric_alarm" "file_rejection_alarm" {
  alarm_name = "FileUploads-HighRejectionRate"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods = "2"
  metric_name = "file.rejections.total"
  namespace = "FileUploadService"
  period = "300"
  statistic = "Sum"
  threshold = "10"
  alarm_description = "This alarm triggers if more than 10 files are rejected in a 10 minute period."
}