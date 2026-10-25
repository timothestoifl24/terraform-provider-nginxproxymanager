resource "nginxproxymanager_letsencrypt_certificate" "example" {
  domain_names             = ["www.example.com", "example.com"]
  dns_challenge            = true
  dns_provider             = "route53"
  dns_provider_credentials = <<EOT
[default]
aws_access_key_id=exampleaccesskey
aws_secret_access_key=examplesecretkey
EOT
  propagation_seconds      = 5
}
