# A policy container within a zone. Cedar rules are managed separately
# through policy versions.
resource "keycard_policy" "admin_access" {
  zone_id     = keycard_zone.production.id
  name        = "admin-access"
  description = "Grants administrative access to protected resources"
}
