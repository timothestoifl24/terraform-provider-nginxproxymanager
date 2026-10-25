# Terraform Provider for NGINX Proxy Manager

> [!IMPORTANT]
> This is a project which is mainly used for my own learning, and is **not** production ready.
> 
> I will try to maintain this project as much as possible but make no guarantees on fixes or
> updates to the provider. 
>
> Please feel free to make contributions and I will try to review them when I can.

## Contents

- [Terraform Provider for NGINX Proxy Manager](#terraform-provider-for-nginx-proxy-manager)
  - [Contents](#contents)
  - [Documentation](#documentation)
  - [Installation](#installation)
  - [Supported Versions](#supported-versions)
  - [Importing an existing NPM](#importing-an-existing-npm)
  - [Contributing](#contributing)
  - [Test Setup](#test-setup)
    - [Quirks](#quirks)

## Documentation

All documentation for this provider can be found on the Terraform Registry.

## Installation

This provider is installed via the following configuration block:

```terraform
terraform {
    required_providers {
        nginxproxymanager = {
            source = "edklesel/nginxproxymanager"
        }
    }
}
```

## Supported Versions

The provider is tested against the following versions in CI:
- 2.15.1
- 2.16.0

Due to API changes, the minimum required version for Nginx Proxy Manager is 2.15.0.

The provider is tested against Terraform versions 1.15.0 and above

## Importing an existing NPM

Every resource can be imported by the ID NPM shows in the UI and API, except `nginxproxymanager_user_permissions`
(the user's ID), `nginxproxymanager_default_site` (always `default-site`) and `nginxproxymanager_user_password`
(NPM never returns passwords, so it can't be imported).

With Terraform 1.5 or later you can generate the config for everything at once. Write `import` blocks for the
objects you want, for example with this script against the NPM API:

```bash
NPM=http://npm.example.com:81
TOKEN=$(curl -s -X POST "$NPM/api/tokens" -H 'Content-Type: application/json' \
  -d '{"identity":"admin@example.com","secret":"changeme"}' | jq -r .token)

for kind in proxy-hosts:proxy_host redirection-hosts:redirection_host dead-hosts:dead_host \
            streams:stream access-lists:access_list; do
  path=${kind%%:*}; type=${kind##*:}
  curl -s -H "Authorization: Bearer $TOKEN" "$NPM/api/nginx/$path" |
    jq -r --arg type "$type" '.[] | "import {\n  to = nginxproxymanager_\($type).\($type)_\(.id)\n  id = \"\(.id)\"\n}\n"'
done > imports.tf
```

Then let Terraform write the resources and check them:

```bash
terraform plan -generate-config-out=generated.tf
terraform apply
terraform plan   # should show no changes
```

NPM doesn't return secrets, so after an import:

- access list user passwords and Letsencrypt `dns_provider_credentials` have to be added to the config.
  The next apply sets them in place, without replacing anything.
- custom certificates need their certificate and key in the config.

## Contributing

For development, you should be able to run the below commands to get a test environment set up. **This required docker compose to be installed.**
```bash
make docker-up
```

To install the provider for local testing:
```bash
make install
```

Prior to creating a PR, please ensure that the following commands have been run to ensure formatting and linting is done.
```bash
make generate
make lint
```

Tests can be run using the relevant command:
```bash
make test-sdk # For SDK testing
make test-acc # For TF provider testing
```

## Test Setup

Tests are set up using the following components, mirroring the setup in the main NGINX Proxy Manager repository:

- `NPM` / `MariaDB` - The main NGINX Proxy Manager service
- `PowerDNS` / `MariaDB` - DNS provider used for ACME DNS challenges
- `Step-CA` - ACME Certificate provider
- `DNSRouter` - Intercept DNS requests for the domain `example.com` and forward to PowerDNS.

### Running the tests with Podman

The test stack also runs under rootless Podman, with two differences: nginx in the NPM container needs to
be allowed to bind to port 80, and `make docker-up` uses GNU `timeout` and `grep -P`, which macOS doesn't have.
Add an override file:

```yaml
# docker/docker-compose.override.yaml
services:
  npm:
    sysctls:
      net.ipv4.ip_unprivileged_port_start: "0"
```

and start the stack directly:

```bash
export NPM_UID=1000 NPM_GID=$(id -g) PWD=$(pwd)
podman compose -f docker/docker-compose.yaml -f docker/docker-compose.override.yaml up -d
podman exec npm bash -c 'chmod g+r /data/keys.json'
TF_ACC=1 go test -count=1 ./internal/sdk ./internal/provider/tests
```

### Quirks

- NPM has access to the step-ca volume as it needs to read the root CA for requests CA certificate verification.
- NPM locally mounts its data directory so the test suite has access to keys.json, which is used to decrypt the JWT tokens in tests.
- Step-CA, for some reason, creates its `root_ca.crt` file with `1000:1000` permissions, regardless of the user running the container. Therefore, the makefile sets the UID to this user.
- Similarly, NPM creates the `keys.json` file with only user rw permissions, so there's a command to allow group read permission during the `docker-up` make command so that it can be read by tests, in case the UID running the tests isn't 1000.
