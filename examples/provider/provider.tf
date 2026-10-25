provider "nginxproxymanager" {
  host     = "localhost:81"
  username = "admin@example.com"
  password = "changeme"
  secure   = false
}

# Or with a token, e.g for a user with two-factor authentication. The token
# can also be set with NGINXPROXYMANAGER_TOKEN, and the other settings with
# NGINXPROXYMANAGER_HOST, NGINXPROXYMANAGER_USERNAME, NGINXPROXYMANAGER_PASSWORD
# and NGINXPROXYMANAGER_SECURE.
provider "nginxproxymanager" {
  alias = "token"
  host  = "npm.example.com"
  token = var.npm_token
}

variable "npm_token" {
  type      = string
  sensitive = true
}
