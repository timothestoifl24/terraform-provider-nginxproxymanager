data "nginxproxymanager_version" "current" {}

output "npm_version" {
  value = data.nginxproxymanager_version.current.version
}
