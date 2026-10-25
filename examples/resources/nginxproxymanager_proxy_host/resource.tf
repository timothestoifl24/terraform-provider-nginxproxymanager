resource "nginxproxymanager_proxy_host" "example" {
  forward_host   = "server.local"
  forward_port   = 8080
  forward_scheme = "https"
  domain_names   = ["example.com", "www.example.com"]

  certificate_id = nginxproxymanager_letsencrypt_certificate.example.id
  ssl_forced     = true
  http2_support  = true
  block_exploits = true
  access_list_id = nginxproxymanager_access_list.example.id

  location {
    path            = "/api"
    forward_scheme  = "http"
    forward_host    = "api.local"
    forward_port    = 9000
    forward_path    = "/v1"
    advanced_config = "proxy_read_timeout 300s;"
  }
}
