variable "aws_region" {
  description = "The AWS region to deploy resources in."
  type        = string
  default     = "ap-southeast-2"
}

variable "app_name" {
  description = "The name of the application."
  type        = string
  default     = "restaurant-service"
}

variable "app_port" {
  description = "The port the application container listens on."
  type        = number
  default     = 2131
}

variable "ingress_cidr_blocks" {
  description = "The list of CIDR blocks to allow for inbound traffic to the security group."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}