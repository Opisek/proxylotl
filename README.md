# Proxelot

Proxelot is a powerful reverse proxy for Minecraft servers, ideal for the following scenarios:
1. You have multiple Minecraft servers and you want to connect to them with hostnames like `survival.example.com` and `creative.example.com` instead of by using different ports.
2. You have one or more Minecraft servers that are not accessible from the internet and you want to proxy user connections to it.
3. You have one or more Minecraft servers that you want to shut down and start up automatically to save resources.

Further down you can find guides on how to install and configure Proxelot.

# Installation
The following section describes different ways to install Proxelot.

The simplest way is to use Docker compose with pre-built images. More advanced users may choose to build Docker images themselves or run Proxelot without Docker.

## Docker Compose (Recommended)
Proxelot is available as a Docker container. The recommended setup is to use a Docker compose file on a Linux server.

1. Make sure [Docker](https://docs.docker.com/engine/install/) is installed.
2. Copy the Docker compose template from the source code or from below and put it on your server, for example at `~/docker/proxelot/docker-compose.yml`.
  ```yml
  services:
    proxelot:
      container_name: proxelot
      image: opisek/proxelot:latest
      volumes:
        - /srv/proxelot/scripts:/app/scripts:ro
        - /srv/proxelot/config.yml:/app/config.yml:ro
        - /etc/localtime:/etc/localtime:ro
        - /etc/timezone:/etc/timezone:ro
      network_mode: host
      environment:
        - CONFIG=./config.yml
        - PORT=25565
      restart: unless-stopped
      build:
        context: .
        dockerfile: Dockerfile
  ```
3. Feel free to adjust the volume mounts and environment variables to fit your setup. The default template assumes a directory `/srv/proxelot` containing the sub-directory `scripts` and the configuration file `config.yml`.
4. Create a configuration file at the path specified in the previous step. You can keep it empty for now. A [configuration guide](#configuration) is available further down.
5. While inside the directory containing the compose file (e.g., `~/docker/proxelot`) start the container:
```sh
docker compose up -d
```

## Building Images (Advanced)
More advanced users may opt for building their own Docker image, either via the help of Docker compose or manually.

1. Make sure [Docker](https://docs.docker.com/engine/install/) is installed.
2. Clone this repository:
```sh
git clone https://github.com/Opisek/proxelot.git
```
3. Enter the cloned repository:
```sh
cd proxelot
```
4. Either build the image using Docker compose:
```sh
docker compose build
```
or use the provided Dockerfile directly:
```
docker build .
```
5. Further set-up is left up to the user.

## Baremetal (Advanced)
More advanced users may opt for running Proxelot without Docker.

1. Make sure [Go](https://go.dev/doc/install) is installed (version 1.26.0 or newer).
2. Clone this repository:
```sh
git clone https://github.com/Opisek/proxelot.git
```
3. Enter the source directory:
```sh
cd proxelot/src
```
4. Build the executable:
```sh
go build
```
5. Further set-up is left up to the user. Typically, you will want to create a system daemon out of the resulting executable `proxelot`.

# Configuration
This section describes how to configure Proxelot and use its various features.

## Upstreams
First let us see how we can configure Proxelot to redirect us to different servers depending on the hostname we use.

Let's say that players joining `survival.example.com` should be connected to `example.com:25566` and players joining `creative.example.com` should be connected to `example.com:25567`. The following configuration is sufficient for this:

```yml
servers:
  survival:
    from:
      - survival.example.com
    to: example.com:25566
  creative:
    from:
      - creative.example.com
    to: example.com:25567
```

Notice that the `from` parameter is a list. Indeed, we could specify multiple hostname and port configurations to redirect us to the same server. For example, `example.com` could connect us to the same server as `survival.example.com`:

```yml
servers:
  survival:
    from:
      - example.com
      - survival.example.com
    to: example.com:25566
  creative:
    from:
      - creative.example.com
    to: example.com:25567
```

## Proxy vs Redirect
There are two distinct ways how a user is connected to an upstream server. By default, Proxelot proxies the connection. This means that all the traffic from the user first go to Proxelot and then to the upstream server.

This is great when you only want to expose one port to the internet:

```yml
servers:
  survival:
    from:
      - survival.example.com
    to: example.com:25566
  creative:
    from:
      - creative.example.com
    to: example.com:25567
```

An alternative is to use "redirects" otherwise known as "transfers. In this case, Proxelot will instruct the client to connect directly to the upstream server:

```yml
servers:
  survival:
    from:
      - survival.example.com
    to: example.com:25566
    redirect: true
  creative:
    from:
      - creative.example.com
    to: example.com:25567
    redirect: true
```

For this to succeed, you must ensure that the upstream server is accessible from the internet, for example by forwarding its port as well.

In this setup, no overhead or latency is introduced, since after connecting to the server, Proxelot is no longer involved in the clients' connections.

While a client is connected to the upstream server, you can even restart or do maintenance on Proxelot without affecting active users' play sessions.

You can mix and match `redirect: false` (default) and `redirect: true` for different servers, depending on your needs.

## Watchdog
You can use the watchdog feature to automatically start your upstream server when players attempt to connect and close it when there are no online players.

This is ideal when you want to save on electricity or other server costs when nobody is actively playing.

The base configuration looks as follows:

```yml
servers:
  survival:
    from:
      - survival.example.com
    to: example.com:25566
    watchdog:
      start: echo "start"
      stop: echo "stop"
      grace: 60
```
Note that you could naturally use `redirect: true` with watchdog as well.

With this configuration, the command `echo "start"` is executed when players attempt to connect to the server.

If there have been no players on the server for over 60 seconds, `echo "stop"` is executed.

Of course, these example commands would not actually do anything. How you configure them will depend entirely on your setup. You could, for example, do one of the following:
1. Use a web API with an authorization token to start and stop a server from a remote hosting provider.
2. Use `ssh` to connect to the host server and start or stop the server as set up.
3. Create corresponding `.sh` scripts in the `scripts` directory and invoke them by setting `start: ./start.sh` and `stop: ./stop.sh`.
4. Start the upstream server using Docker.

You must, however, remember, that Proxelot (by default) runs in a dockerized environment. This means, that you cannot invoke scripts or commands on your host system by default, as they are isolated to the Proxelot Docker container.

This can be overcome either by running Proxelot on baremetal (for more advanced users), or by setting up a named pipe and a helper daemon on the host server.

For example, if you have a Docker compose file at path `~/docker/minecraft/docker-compose.yml` with two containers called `survival` and `creative` and you want Proxelot to be able to start and stop them, your setup might look as follows:

1. Create a named pipe at `/srv/proxelot/scripts` (or wherever you mounted your `scripts` volume):
```sh
sudo mkfifo /srv/proxelot/scripts/pipe
```
2. Set your configuration file as follows:
```yml
servers:
  survival:
    from:
      - survival.example.com
    to: example.com:25566
    watchdog:
      start: echo "start survival" >> ./scripts/pipe
      stop: echo "stop survival" >> ./scripts/pipe
      grace: 60
  creative:
    from:
      - creative.example.com
    to: example.com:25567
    watchdog:
      start: echo "start creative" >> ./scripts/pipe
      stop: echo "stop creative" >> ./scripts/pipe
      grace: 60
```
3. Create a daemon script, for example at `~/scripts/proxelot-helper.sh`:
```sh
#!/bin/bash

PIPE_PATH=/srv/proxelot/scripts/pipe
COMPOSE_PATH=/home/YOUR_USERNAME_HERE/docker/minecraft/docker-compose.yml

if [[ ! -p $PIPE_PATH ]]; then
  mkfifo $PIPE_PATH
fi

while true; do
  CMD=`cat ${PIPE_PATH}`
  PARTS=($CMD)

  if [[ "${#PARTS[@]}" -ne 2 ]]; then
    continue
  fi

  case ${PARTS[0]} in
    start)
      docker compose -f $COMPOSE_PATH up -d ${PARTS[1]}
    ;;
    stop)
      docker compose -f $COMPOSE_PATH stop ${PARTS[1]}
    ;;
  esac
done
```
Remember to fill in `YOUR_USERNAME_HERE`.

4. Set up the daemon service. On systemd-based distributions (like Debian or Ubuntu) this can be done by creating the file `/etc/systemd/system/proxelot-helper.service`:
```sh
[Unit]
Description=Proxelot helper service for starting and stopping containers

[Service]
Type=simple
User=YOUR_USERNAME_HERE
ExecStart=/home/YOUR_USERNAME_HERE/scripts/proxelot-helper.sh
Restart=on-failure

[Install]
WantedBy=default.target
```

Remember to fill in `YOUR_USERNAME_HERE`.

5. Start your daemon service. On systemd-based distributions this can be done as follows|:
```sh
sudo systemctl daemon-reload
sudo systemctl enable proxelot-helper
sudo systemctl start proxelot-helper
```