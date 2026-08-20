terraform {
  backend "s3" {}
}

terraform {
  required_version = ">= 1.3.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      project = var.project_name
    }
  }
}

# RESOURCES

data "aws_caller_identity" "current" {}
