resource "nginxproxymanager_proxy_host" "example" {
  domain_names = ["domain123.com"]

  http2_support  = true
  certificate_id = 42
}