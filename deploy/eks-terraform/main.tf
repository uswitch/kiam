terraform {
  required_version = "~> 0.11.14"
}

provider "aws" {
  profile = "${var.profile}"
  region  = "${var.region}"

  version = "~> 2.14.0"

  assume_role {
    role_arn = "${var.assumed_role}"
  }
}

provider "template" {
  version = "1.0.0"
}

locals {
  # Default tags
  default_tags = "${map(
    "Origin", "Terraform"
  )}"
}
