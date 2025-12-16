variable "vpc_id" {
  description = "The ID of the VPC where the PrivateLink endpoint will be created"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs where the VPC endpoint will be available"
  type        = list(string)
}

variable "aws_region" {
  description = "AWS region where resources will be created"
  type        = string
  default     = "us-east-1"
}

variable "security_group_ids" {
  description = "List of security group IDs to associate with the VPC endpoint. If not provided, the default VPC security group will be used."
  type        = list(string)
  default     = []
}
