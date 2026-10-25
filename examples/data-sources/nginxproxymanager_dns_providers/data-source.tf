data "nginxproxymanager_dns_providers" "all" {}

# The credentials format NPM expects for Cloudflare
output "cloudflare_credentials" {
  value = one([for p in data.nginxproxymanager_dns_providers.all.providers : p.credentials if p.id == "cloudflare"])
}
