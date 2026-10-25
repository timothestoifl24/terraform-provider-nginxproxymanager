# Forward TCP connections on port 5432 to a database
resource "nginxproxymanager_stream" "example" {
  incoming_port   = 5432
  forwarding_host = "db.local"
  forwarding_port = 5432
  tcp_forwarding  = true
}
