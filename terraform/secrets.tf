
# Creates a secret in AWS Secrets Manager to store the database password.
resource "aws_secretsmanager_secret" "db_password" {
  name = "${var.app_name}/db_password"
  description = "Database password for the file uploader application"

  recovery_window_in_days = 0
}

# Creates a version of the secret with a placeholder value.
# The actual password will be managed outside of Terraform.
resource "aws_secretsmanager_secret_version" "db_password_version" {
  secret_id     = aws_secretsmanager_secret.db_password.id
  secret_string = "changeme" # Placeholder value
}
