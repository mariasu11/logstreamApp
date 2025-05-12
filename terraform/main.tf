# Terraform configuration for deploying LogStream to AWS

provider "aws" {
  region = var.aws_region
}

# S3 bucket for storing logs
resource "aws_s3_bucket" "logstream_logs" {
  bucket = "${var.prefix}-logstream-logs"

  tags = {
    Name        = "${var.prefix}-logstream-logs"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Enable versioning for the bucket
resource "aws_s3_bucket_versioning" "logs_versioning" {
  bucket = aws_s3_bucket.logstream_logs.id
  versioning_configuration {
    status = "Enabled"
  }
}

# Server-side encryption for the bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "logs_encryption" {
  bucket = aws_s3_bucket.logstream_logs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# VPC for LogStream deployment
resource "aws_vpc" "logstream_vpc" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "${var.prefix}-logstream-vpc"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Public subnets
resource "aws_subnet" "public" {
  count                   = length(var.availability_zones)
  vpc_id                  = aws_vpc.logstream_vpc.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name        = "${var.prefix}-logstream-public-${count.index + 1}"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Internet gateway
resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.logstream_vpc.id

  tags = {
    Name        = "${var.prefix}-logstream-igw"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Route table for public subnets
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.logstream_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }

  tags = {
    Name        = "${var.prefix}-logstream-public-rt"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Associate route table with public subnets
resource "aws_route_table_association" "public" {
  count          = length(aws_subnet.public)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Security group for LogStream
resource "aws_security_group" "logstream" {
  name        = "${var.prefix}-logstream-sg"
  description = "Security group for LogStream"
  vpc_id      = aws_vpc.logstream_vpc.id

  # API access
  ingress {
    from_port   = 8000
    to_port     = 8000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidr_blocks
  }

  # All outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.prefix}-logstream-sg"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# IAM role for EC2 instance
resource "aws_iam_role" "logstream_role" {
  name = "${var.prefix}-logstream-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

# IAM policy for S3 access
resource "aws_iam_policy" "s3_access" {
  name        = "${var.prefix}-logstream-s3-access"
  description = "Allow LogStream to access S3 bucket"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "s3:ListBucket",
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Effect = "Allow"
        Resource = [
          aws_s3_bucket.logstream_logs.arn,
          "${aws_s3_bucket.logstream_logs.arn}/*"
        ]
      }
    ]
  })
}

# Attach S3 policy to role
resource "aws_iam_role_policy_attachment" "s3_access" {
  role       = aws_iam_role.logstream_role.name
  policy_arn = aws_iam_policy.s3_access.arn
}

# IAM instance profile
resource "aws_iam_instance_profile" "logstream_profile" {
  name = "${var.prefix}-logstream-profile"
  role = aws_iam_role.logstream_role.name
}

# EC2 instance for LogStream
resource "aws_instance" "logstream" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.logstream.id]
  subnet_id              = aws_subnet.public[0].id
  iam_instance_profile   = aws_iam_instance_profile.logstream_profile.name

  root_block_device {
    volume_size = 20
    volume_type = "gp2"
  }

  user_data = <<-EOF
              #!/bin/bash
              mkdir -p /opt/logstream/logs
              mkdir -p /opt/logstream/plugins
              
              # Install LogStream
              aws s3 cp s3://${var.prefix}-logstream-deployment/logstream /usr/local/bin/logstream
              chmod +x /usr/local/bin/logstream
              
              # Create service file
              cat > /etc/systemd/system/logstream.service << 'SERVICEFILE'
              [Unit]
              Description=LogStream Service
              After=network.target
              
              [Service]
              Type=simple
              User=root
              ExecStart=/usr/local/bin/logstream serve --host 0.0.0.0 --port 8000 --storage disk --storage-path /opt/logstream/logs
              Restart=always
              RestartSec=5
              
              [Install]
              WantedBy=multi-user.target
              SERVICEFILE
              
              # Enable and start the service
              systemctl enable logstream
              systemctl start logstream
              EOF

  tags = {
    Name        = "${var.prefix}-logstream"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Elastic IP for LogStream instance
resource "aws_eip" "logstream" {
  domain = "vpc"

  tags = {
    Name        = "${var.prefix}-logstream-eip"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Associate Elastic IP with instance
resource "aws_eip_association" "logstream" {
  instance_id   = aws_instance.logstream.id
  allocation_id = aws_eip.logstream.id
}

# CloudWatch Log Group for LogStream
resource "aws_cloudwatch_log_group" "logstream" {
  name              = "/logstream/${var.prefix}"
  retention_in_days = 30

  tags = {
    Name        = "${var.prefix}-logstream-logs"
    Environment = var.environment
    Project     = "LogStream"
  }
}

# Output important information
output "logstream_ip" {
  description = "The public IP address of the LogStream instance"
  value       = aws_eip.logstream.public_ip
}

output "logs_bucket" {
  description = "The S3 bucket for LogStream logs"
  value       = aws_s3_bucket.logstream_logs.bucket
}

output "api_endpoint" {
  description = "The LogStream API endpoint"
  value       = "http://${aws_eip.logstream.public_ip}:8000"
}
