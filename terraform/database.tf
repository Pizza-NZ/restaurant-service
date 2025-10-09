resource "aws_db_subnet_group" "default" {
  name       = "${var.app_name}-db-subnet-group"
  subnet_ids = [aws_subnet.private_a.id, aws_subnet.private_b.id]

  tags = {
    Name = "${var.app_name} DB Subnet Group"
  }
}

resource "aws_rds_cluster" "default" {
  cluster_identifier      = "${var.app_name}-aurora-cluster"
  engine                  = "aurora-postgresql"
  engine_version          = "13.7"
  database_name           = "restaurant"
  master_username         = "postgres"
  master_password         = "yoursecurepassword" # <-- Use AWS Secrets Manager for this in a real project!
  db_subnet_group_name    = aws_db_subnet_group.default.name
  vpc_security_group_ids  = [aws_security_group.ecs_service.id] # For simplicity, reuse SG. Better to have a dedicated one.
  skip_final_snapshot     = true
}

resource "aws_rds_cluster_instance" "default" {
  count              = 1 # Start with one instance
  identifier         = "${var.app_name}-instance-${count.index}"
  cluster_identifier = aws_rds_cluster.default.id
  instance_class     = "db.t3.medium"
  engine             = aws_rds_cluster.default.engine
  engine_version     = aws_rds_cluster.default.engine_version
}