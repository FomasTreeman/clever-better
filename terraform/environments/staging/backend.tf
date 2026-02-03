terraform {
  backend "s3" {
    bucket         = "clever-better-terraform-state-staging"
    key            = "terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "clever-better-terraform-locks"
    encrypt        = true
  }
}
