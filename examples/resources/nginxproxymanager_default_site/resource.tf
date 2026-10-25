# Redirect requests for hosts NPM doesn't know
resource "nginxproxymanager_default_site" "example" {
  page     = "redirect"
  redirect = "https://www.example.com"
}
