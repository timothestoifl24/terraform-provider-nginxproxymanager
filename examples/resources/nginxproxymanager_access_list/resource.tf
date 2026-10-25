resource "nginxproxymanager_access_list" "example" {
  name = "test12345"

  item {
    username = "BasicAuthUser"
    password = "S3cur3P455w0rd!"
  }
  client {
    address   = "1.2.3.4"
    directive = "deny"
  }
}