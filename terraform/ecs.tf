# Creates an ECS cluster to run the application.
resource "aws_ecs_cluster" "main" {
  name = "${var.app_name}-cluster"

  tags = {
    Name = "${var.app_name}-cluster"
  }
}

# Creates a security group for the ECS service.
# This allows inbound traffic on the application port and outbound traffic to the internet.
resource "aws_security_group" "ecs_service" {
  name        = "${var.app_name}-ecs-service-sg"
  description = "Allow inbound traffic to the ${var.app_name} service"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = var.app_port
    to_port     = var.app_port
    protocol    = "tcp"
    cidr_blocks = var.ingress_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.app_name}-ecs-service-sg"
  }
}

# Creates the ECS task definition for the application.
# This defines the container image, CPU/memory, IAM roles, and logging configuration.
resource "aws_ecs_task_definition" "main" {
  family                   = var.app_name
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_role.arn

  container_definitions = jsonencode([
    {
      name      = "order-service"
      image     = "${aws_ecr_repository.main.repository_url}:latest" # Placeholder, will be updated by CI/CD
      essential = true
      portMappings = [
        {
          containerPort = var.app_port
          hostPort      = var.app_port
        }
      ]
      environment = [
        {
          name  = "GIN_MODE",
          value = "release"
        },
        {
          name  = "APP_ENV",
          value = "production"
        },
        {
          name  = "NATS_URL",
          value = "nats://localhost:4222"
        },
        {
          name  = "DB_HOST",
          value = aws_rds_cluster.default.endpoint
        }
      ]
      secrets = [
        {
          name      = "DB_PASSWORD",
          valueFrom = aws_secretsmanager_secret.db_password.arn
        }
      ]
      logConfiguration = {
        logDriver = "awslogs",
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.main.name,
          "awslogs-region"        = var.aws_region,
          "awslogs-stream-prefix" = "order-service"
        }
      }
    },
    {
      name      = "printer-service"
      image     = "${aws_ecr_repository.main.repository_url}:latest" # Placeholder, will be updated by CI/CD
      essential = true
      environment = [
        {
          name  = "NATS_URL",
          value = "nats://localhost:4222"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs",
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.main.name,
          "awslogs-region"        = var.aws_region,
          "awslogs-stream-prefix" = "printer-service"
        }
      }
    },
    {
      name      = "nats"
      image     = "nats:latest"
      essential = true
      portMappings = [
        {
          containerPort = 4222,
          hostPort      = 4222
        }
      ]
      logConfiguration = {
        logDriver = "awslogs",
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.main.name,
          "awslogs-region"        = var.aws_region,
          "awslogs-stream-prefix" = "nats"
        }
      }
    }
  ])

  tags = {
    Name = "${var.app_name}-task"
  }
}

# Creates the ECS service to run the application.
# This service ensures that the specified number of tasks are running and registers them with the load balancer.
resource "aws_ecs_service" "main" {
  name            = "${var.app_name}-service"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.main.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_groups = [aws_security_group.ecs_service.id]
    assign_public_ip = false # The service runs in private subnets
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.main.arn
    container_name   = var.app_name
    container_port   = var.app_port
  }

  depends_on = [
    aws_iam_role_policy_attachment.ecs_task_role_secrets_access,
  ]

  tags = {
    Name = "${var.app_name}-service"
  }
}