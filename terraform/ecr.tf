
# Creates an Elastic Container Registry (ECR) repository to store the application's Docker images.
resource "aws_ecr_repository" "main" {
  name = var.app_name
  image_tag_mutability = "MUTABLE"

  force_delete = true

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name = "${var.app_name}-repo"
  }
}
