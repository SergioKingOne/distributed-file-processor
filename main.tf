terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = var.region
}

module "input_bucket" {
  source        = "./modules/s3_bucket"
  bucket_name   = "dfp-input-bucket"
  force_destroy = true
}

module "file_processing_topic" {
  source     = "./modules/sns_topic"
  topic_name = "file-processing-topic"
}

module "worker_queue" {
  source     = "./modules/sqs_queue"
  queue_name = "worker-queue"
}

module "chunker_lambda" {
  source           = "./modules/lambda_function"
  function_name    = "chunker-lambda"
  handler          = "chunker"
  runtime          = "provided.al2"
  filename         = "bin/chunker.zip"
  source_code_hash = filebase64sha256("bin/chunker.zip")
  environment_variables = {
    SNS_TOPIC_ARN = module.file_processing_topic.topic_arn
    CHUNK_SIZE    = "10485760" # 10 MB in bytes
    BUCKET_NAME   = module.input_bucket.bucket_id
  }
  timeout     = 60
  memory_size = 256
}

module "output_bucket" {
  source        = "./modules/s3_bucket"
  bucket_name   = "dfp-output-bucket"
  force_destroy = true
}

module "worker_lambda" {
  source           = "./modules/lambda_function"
  function_name    = "worker-lambda"
  handler          = "worker"
  runtime          = "provided.al2"
  filename         = "bin/worker.zip"
  source_code_hash = filebase64sha256("bin/worker.zip")
  environment_variables = {
    SQS_QUEUE_URL = module.worker_queue.queue_id
    OUTPUT_BUCKET = module.output_bucket.bucket_id
  }
  timeout     = 60
  memory_size = 256
}

# Add Lambda permission for S3
resource "aws_lambda_permission" "allow_bucket" {
  statement_id  = "AllowS3Invoke"
  action        = "lambda:InvokeFunction"
  function_name = module.chunker_lambda.lambda_function_name
  principal     = "s3.amazonaws.com"
  source_arn    = module.input_bucket.bucket_arn
}

# Add S3 bucket notification configuration
resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = module.input_bucket.bucket_id

  lambda_function {
    lambda_function_arn = module.chunker_lambda.lambda_function_arn
    events              = ["s3:ObjectCreated:*"]
  }

  depends_on = [aws_lambda_permission.allow_bucket]
}

resource "aws_sns_topic_subscription" "worker_queue_subscription" {
  topic_arn = module.file_processing_topic.topic_arn
  protocol  = "sqs"
  endpoint  = module.worker_queue.queue_arn
}

resource "aws_sqs_queue_policy" "worker_queue_policy" {
  queue_url = module.worker_queue.queue_id

  policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
    {
        "Sid": "Allow-SendMessage-From-SNS",
        "Effect": "Allow",
        "Principal": {
            "Service": "sns.amazonaws.com"
        },
        "Action": "sqs:SendMessage",
        "Resource": "${module.worker_queue.queue_arn}",
        "Condition": {
            "ArnEquals": {
                "aws:SourceArn": "${module.file_processing_topic.topic_arn}"
            }
        }
    }
    ]
}
POLICY
}

# Add SQS permission for Worker Lambda
resource "aws_lambda_permission" "allow_sqs" {
  statement_id  = "AllowSQSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = module.worker_lambda.lambda_function_name
  principal     = "sqs.amazonaws.com"
  source_arn    = module.worker_queue.queue_arn
}
