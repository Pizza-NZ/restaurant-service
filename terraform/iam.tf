
# IAM role that allows ECS tasks to be executed.
# This role is used by the ECS agent to start and stop containers.
resource "aws_iam_role" "ecs_task_execution_role" {
  name = "${var.app_name}-ecs-task-execution-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })
}

# Attaches the AmazonECSTaskExecutionRolePolicy to the ECS task execution role.
# This policy provides the necessary permissions for ECS to manage tasks.
resource "aws_iam_role_policy_attachment" "ecs_task_execution_role_policy" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# IAM role for the ECS task itself.
# This role is used by the application running inside the container.
resource "aws_iam_role" "ecs_task_role" {
  name = "${var.app_name}-ecs-task-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })
}

# IAM policy that allows the ECS task to access the database password secret.
resource "aws_iam_policy" "secrets_access" {
  name        = "${var.app_name}-secrets-access"
  description = "Allow access to the database password secret"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Effect   = "Allow"
        Resource = aws_secretsmanager_secret.db_password.arn
      }
    ]
  })
}

# Attaches the secrets access policy to the ECS task role.
resource "aws_iam_role_policy_attachment" "ecs_task_role_secrets_access" {
  role       = aws_iam_role.ecs_task_role.name
  policy_arn = aws_iam_policy.secrets_access.arn
}

# Attaches the secrets access policy to the ECS task execution role.
# This is required for the ECS agent to pull the secret and inject it into the container.
resource "aws_iam_role_policy_attachment" "ecs_task_execution_role_secrets_access" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = aws_iam_policy.secrets_access.arn
}

resource "aws_iam_policy" "observability_policy" {
  name = "${var.app_name}-observability-policy"
  description = "IAM policy for OTel collector to send data to AWS Observability"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [ 
          "xray:PutTraceSegments",
          "xray:PutTelemetryRecords"
        ],
        Resource = "*"
      },
      {
        Effect = "Allow",
        Action = "cloudwatch:PutMetricData",
        Resource = "*"
      },
      {
        Effect = "Allow",
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ],
        Resource = "arn:aws:logs:*:*:log-group:/ecs/${var.app_name}-*:*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "task_observability_attachment" {
  role = aws_iam_role.ecs_task_role.name
  policy_arn = aws_iam_policy.observability_policy.arn
}