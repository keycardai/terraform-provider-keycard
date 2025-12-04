# VPC Endpoint for Keycard's VPC Endpoint Service
resource "aws_vpc_endpoint" "keycard" {
  vpc_id              = var.vpc_id
  service_name        = "com.amazonaws.vpce.us-east-1.vpce-svc-046268684906e4f69"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = var.subnet_ids
  security_group_ids  = length(var.security_group_ids) > 0 ? var.security_group_ids : null
  private_dns_enabled = false

  tags = {
    Name = "keycard-privatelink"
  }
}

# Route53 Private Hosted Zone for api.keycard.ai
resource "aws_route53_zone" "keycard_api" {
  name = "api.keycard.ai"

  vpc {
    vpc_id = var.vpc_id
  }

  tags = {
    Name = "keycard-ai-privatelink"
  }
}

# ALIAS record pointing api.keycard.ai to the VPC endpoint
resource "aws_route53_record" "keycard_api" {
  zone_id = aws_route53_zone.keycard_api.zone_id
  name    = "api.keycard.ai"
  type    = "A"

  alias {
    name                   = aws_vpc_endpoint.keycard.dns_entry[0].dns_name
    zone_id                = aws_vpc_endpoint.keycard.dns_entry[0].hosted_zone_id
    evaluate_target_health = false
  }
}

# Route53 Private Hosted Zone for console.keycard.ai
resource "aws_route53_zone" "keycard_console" {
  name = "console.keycard.ai"

  vpc {
    vpc_id = var.vpc_id
  }

  tags = {
    Name = "keycard-console-privatelink"
  }
}

# ALIAS record pointing console.keycard.ai to the VPC endpoint
resource "aws_route53_record" "keycard_console" {
  zone_id = aws_route53_zone.keycard_console.zone_id
  name    = "console.keycard.ai"
  type    = "A"

  alias {
    name                   = aws_vpc_endpoint.keycard.dns_entry[0].dns_name
    zone_id                = aws_vpc_endpoint.keycard.dns_entry[0].hosted_zone_id
    evaluate_target_health = false
  }
}

# Route53 Private Hosted Zone for keycard.cloud (with wildcard)
resource "aws_route53_zone" "keycard_cloud" {
  name = "keycard.cloud"

  vpc {
    vpc_id = var.vpc_id
  }

  tags = {
    Name = "keycard-cloud-privatelink"
  }
}

# Wildcard CNAME record pointing *.keycard.cloud to the VPC endpoint
resource "aws_route53_record" "keycard_cloud_wildcard" {
  zone_id = aws_route53_zone.keycard_cloud.zone_id
  name    = "*.keycard.cloud"
  type    = "CNAME"
  ttl     = 300
  records = [aws_vpc_endpoint.keycard.dns_entry[0].dns_name]
}
