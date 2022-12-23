# SOLDR

SOLDR is an Endpoint Detection and Response system which consists of centralised management part with extensive Web UI and Agents being installed on endpoint systems. SOLDR allows you not only to configure security policies but also write your own modules and make detection of the comprehensive security events as well as do almost instant response on the security alarms.

## Repository structure

- `build` - storage of all built artifacts
- `cmd` - services entry points
- `db` - database migrations
- `internal` - services implementation
- `scripts` - supplemental scripts
- `security` - security configuration
- `web` - web interface source

## Local run

### Prerequisites

- [docker-ce](https://docs.docker.com/engine/installation)
- [docker-compose](https://docs.docker.com/compose)
- [protobuf](https://github.com/protocolbuffers/protobuf/releases)
- [go](https://go.dev/dl/)
- `protoc-gen-go`
    - install with `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
- [node](https://github.com/nodesource/distributions/blob/master/README.md), recommended node 16 LTS
- `python2`
- `mysql` client
- [minio client](https://docs.min.io/minio/baremetal/reference/minio-mc.html)
    - Note: better install as `mcli` due to [conflict](https://github.com/minio/mc/blob/RELEASE.2018-01-18T21-18-56Z/CONFLICT.md)

#### Example: Ubuntu 22.04

```bash
sudo apt update
sudo apt install build-essential ca-certificates openssl software-properties-common curl git mysql-client jq

# golang 1.19
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get install golang-go

# node 16 LTS
curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -
sudo apt-get install -y nodejs

# yarn
curl -sL https://dl.yarnpkg.com/debian/pubkey.gpg | gpg --dearmor | sudo tee /usr/share/keyrings/yarnkey.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/yarnkey.gpg] https://dl.yarnpkg.com/debian stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
sudo apt-get update && sudo apt-get install yarn

# docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
usermod -aG docker "$USER"
newgrp docker
```

### Configure environment

Setup an environment configuration:

```bash
cp .env.template .env
```

### Run whole project by docker compose

Docker images stored in [vxcontrol](https://hub.docker.com/u/vxcontrol) docker hub account.
This scenario requires `.env` and `docker-compose.yml` files.

#### Download and update images

```bash
docker compose pull
```

#### Run services

```bash
docker compose up -d
```

#### Run observability stack components

```bash
docker compose --profile obs up -d
```

#### Stop services

```bash
docker compose down
```

#### Remove services and collected data

```bash
docker compose down -v
```

### First run without compose

SOLDR require prepared DB schema and additional data stored in S3 for proper initialization. Pull dependencies and start MySQL and Minio servers for simplicity using docker.

```bash
docker compose pull
docker compose up -d mysql minio
```

Wait until MySQL and Minio starts (approx. 30 seconds for the first run), and seed required data:

```bash
make db-init
make s3-init
```

Generate certificates and other crypto materials:

**Attention**: Regeneration of crypto configuration will lead to communication breakage for already stored modules and running agents after rebuild the server components.

```bash
make generate-all
```

Build services and a Web UI:

```bash
make build-all
```

In case of some issues you can use build with prepared environment in docker image:

```bash
make build-backend-vxbuild
```

Upload agent binary into binary storage and register it:

```
make s3-upload-vxagent
```

Configure web UI proxy:

```bash
make setup-web-proxy
```

Start a web UI:

```bash
make run-web
```

An API Server:

```bash
make run-api
```

An Agent Server:

```bash
make run-server
```

An Agent instance:

```bash
make run-agent
```

And finally, upload SOLDR modules to local storage:

```bash
docker compose up -d modules
```

### First login and validation of the setup

Open [`http://localhost`](http://localhost) in a browser. Follow the login process and use default credentials:  `admin` / `admin`.

Click on `Modules` tab in the top menu bar, then open module creation wizard using `Create a module` button.
Name your first module and select `Basic` template, rest leave as is.

Click on `Create` button, which will lead you to a Module editor. Now, when a module is created, let's assign it to a policy.

Open `Groups of agents` tab in the top menu bar, then click `Create a group` button and give some name to a new group.

To assign an Agent to a new group, open `Agents` tab in the top menu, select `All agents` filter group, then click on an agent name in the list and on the `Move to group` button in the top right corner of the interface. Finally, select the group that you have just created and finish the process using button `Move`.

Now we need to create a policy that will contain our new module and assign it to that group. To create a policy, open `Policies` tab in the top menu bar, then click on `Create a policy` button, name your first policy and proceed with `Create` button.
Using `Link to groups` button in the top right corner of the interface, create a link between a group and a policy - click on a link icon in the list right near the name of the group that you want to link.

Let's then add the first created module to the policy - Being in the `Policices` tab, you should see `Modules` section with `Available to be added` group being selected by default. Click on `Install` button to the right of the module name in the list.

As a final step, let's validate that the module has been installed on the Agent according to assigned policy and that this module can successfully communicate with other parts of the system. Open `Agents` tab, then get into a module by clicking on its name. In the `Modules` section you will find one module being installed on an agent and information that this module is configured by a policy. Using the gear-looking button to the right of the module name to get into the interactive interface of the module.
Then click on `Send data` button and check the log right below - it should contain following log lines which tells that test data was sent to the agent part of the module and as a result some data was returned:
```
    2:04:20 PM.152 SEND DATA: {"type":"hs_browser","data":"test test test"}
    2:04:20 PM.166 RECV DATA: {"data":"pong","type":"hs_server"}
```

Congratulations! The setup of the SOLDR project is done, it is fully functioning and ready for you to dig into the wilderness of the Endpoint Detection and Response!

### Next steps

Check the configuration in the `.env` file and in `web/proxy.conf.json` to understand the service's configuration, so you can start them using your preferred installation topology.

For the regular start-up of the services, just start an API server, an Agent server and a Web UI.

### Run in debugger in Goland

- Install [EnvFile](https://github.com/ashald/EnvFile) plugin, which will add ability to include file in Run configuration.
- Open `Edit configuratons` -> `Go build`, select `Run Kind` - `Package`. Add package path, for example `soldr/cmd/api`
- Open EnvFile tab, enable it and add `.env` file to file paths.
- Run

### Run in debugger in VSCode

Use `.vscode/launch.json`

Launch on of the available debug tasks:
- launch vxapi
- launch vxserver
- launch vxagent
- launch web ui

### Observability stack

Observability stack collect metrics, traces, logs from SOLDR components. Stack consist of:
- `Grafana` - querying and visualizing observability data
- `VictoriaMetrics` - datastore for server and SOLDR components metrics
- `node-exporter` and `elasticsearch-exporter` - scraping metrics
- `Jaeger` - storing and querying traces
- `Elasticsearch` - datastore for `Jaeger`
- `OpenTelemetry collector` - single entry point to receive, process and export all observability data 

> For more information about collector, visit [`https://opentelemetry.io/docs/collector`](https://opentelemetry.io/docs/collector).

Run observability stack components and then open `Grafana` in a browser [`https://localhost:3000`](https://localhost:3000). Default credentials: `admin/admin`. After default password for `admin` user changed, you can check provisioned SOLDR dashboards by click `Dashboards` icon on menu bar. On this SOLDR dashboards you can view server, agents and modules resource utilization, events statistics etc. 
For checking traces you need to click `Explore` icon on menu bar, then choose `Jaeger` data source from dropdown in the top left. Now you can query and filter traces by `Service Name`, `Operation Name`, `Tags`, `duration` and `Time ranges`.

> Full observability stack requires more resources compare to clean SOLDR.

### Clean up the project

Remove all build files and other security keys:

**Attention**: Removing of crypto configuration and keys will lead to communication breakage for already stored modules and running agents after rebuild the server components.

```bash
make clean-all
```

Only artefacts from build directory (golang services binaries):

```bash
make clean-build
```

Only artefacts from web directory (node-modules and result of building frontend):

```bash
make clean-web
```

Dangerous step to remove security files (use it in edge case):

```bash
make clean-sec
```
