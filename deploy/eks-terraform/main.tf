terraform {
  required_version = "~> 0.11.13"
}

provider "aws" {
  profile = "${var.profile}"
  region  = "${var.region}"

  version = "~> 2.2.0"

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
