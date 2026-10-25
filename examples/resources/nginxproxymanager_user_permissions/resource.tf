resource "nginxproxymanager_user" "example" {
  name     = "test1"
  nickname = "test_nickname yes"
  email    = "test2@test.com"
  roles    = ["user"]
}

resource "nginxproxymanager_user_permissions" "example" {
  user_id = nginxproxymanager_user.test.id
  permissions {
    visibility        = "all"
    proxy_hosts       = "hidden"
    redirection_hosts = "view"
  }
}