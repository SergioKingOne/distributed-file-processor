terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-east-1" # Choose your desired region
}

module "input_bucket" {
  source        = "./modules/s3_bucket"
  bucket_name   = "your-input-bucket-name" # Replace with a unique bucket name
  force_destroy = true
}
