variable "profile" {
  description = "The AWS profile to use."
}

variable "assumed_role" {
  description = "The role to assume when running the Terraform code."
}

variable "cluster_name" {
  description = "The cluster name."
}

variable "cluster_version" {
  description = "The version of the EKS cluster."
}

variable "extra_security_groups" {
  type        = "list"
  description = "Extra security groups to attach to the worker nodes."
}

variable "region" {
  description = "The AWS region where the EKS cluster should be deployed."
}

variable "vpc_id" {
  description = "The id of the vpc to deploy the cluster in."
}

variable "private_subnet_ids" {
  type        = "list"
  description = "A list of private subnets to use for the cluster."
}

variable "public_subnet_ids" {
  type        = "list"
  description = "A list of public subnets to use for the cluster."
}
