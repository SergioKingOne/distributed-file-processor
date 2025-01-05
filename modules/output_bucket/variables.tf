variable "bucket_name" {
  description = "Name of the S3 bucket"
  type        = string
}

variable "acl" {
  description = "ACL for the S3 bucket"
  type        = string
  default     = "private"
}

variable "force_destroy" {
  description = "Whether to force destroy the bucket on deletion"
  type        = bool
  default     = false
}
