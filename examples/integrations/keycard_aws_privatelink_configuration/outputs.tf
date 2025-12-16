output "vpc_endpoint_id" {
  description = "The ID of the VPC endpoint"
  value       = aws_vpc_endpoint.keycard.id
}

output "vpc_endpoint_dns_name" {
  description = "The DNS name of the VPC endpoint"
  value       = aws_vpc_endpoint.keycard.dns_entry[0].dns_name
}

output "route53_zone_api_id" {
  description = "The ID of the Route53 private hosted zone for api.keycard.ai"
  value       = aws_route53_zone.keycard_api.zone_id
}

output "route53_zone_console_id" {
  description = "The ID of the Route53 private hosted zone for console.keycard.ai"
  value       = aws_route53_zone.keycard_console.zone_id
}

output "route53_zone_cloud_id" {
  description = "The ID of the Route53 private hosted zone for keycard.cloud"
  value       = aws_route53_zone.keycard_cloud.zone_id
}
