![screenshot](https://user-images.githubusercontent.com/1033514/64239393-bdbb6480-cf32-11e9-9269-d8d10e7c0dc7.png)

![Build Status](https://github.com/boypt/simple-game/workflows/Go/badge.svg) 

**SimpleGame** is a a self-hosted remote game client, written in Go (golang). Started games remotely, download sets of files on the local disk of the server, which are then retrievable or streamable via HTTP.

This project is a re-branded fork of [cloud-game](https://github.com/jpillora/cloud-game) by `jpillora`.

# Features

* Individual file download control (1.1.3+)
* Run external program on tasks completion: `DoneCmd`
* Stops task when seeding ratio reached: `SeedRatio`
* Download/Upload speed limiter: `UploadRate`/`DownloadRate`
* Detailed transfer stats in web UI.
* [Game Watcher](https://github.com/boypt/simple-game/wiki/Game-Watcher)
* K8s/docker health-check endpoint `/healthz`
* Extra trackers from external source
* Protocol Handler to `magnet:`
* Magnet RSS subscribing supported
* Flexible config file accepts multiple formats (.json/.yaml/.toml) ([by spf13/Viper](https://github.com/spf13/viper/)) (1.2.0+)

Also:
* Single binary
* Cross platform
* Embedded game search
* Real-time updates
* Mobile-friendly
* Fast [content server](http://golang.org/pkg/net/http/#ServeContent)
* IPv6 out of the box
* Updated game engine from [anacrolix/game](https://github.com/anacrolix/game)

# Install

## Binary

See [the latest release](https://github.com/boypt/cloud-game/releases/latest) or use the oneline script to do a quick install on a modern Linux machines.

``` bash
bash <(wget -qO- https://git.io/simplegameqs)
```

The script installs a systemd unit (under `scripts/cloud-game.service`) as service. Read further intructions: [Auth And Security](https://github.com/boypt/simple-game/wiki/AuthSecurity)

If hope to specify a version, just append the version number to the command.

``` bash
bash <(wget -qO- https://git.io/simplegameqs) 1.3.3
```

## Docker [![Docker Pulls](https://img.shields.io/docker/pulls/boypt/cloud-game.svg)][dockerhub]

[dockerhub]: https://hub.docker.com/r/boypt/cloud-game/

``` bash
$ docker run -d -p 3000:3000 -v /path/to/my/downloads:/downloads -v /path/to/my/games:/games boypt/cloud-game
```
When running as a container, keep in mind:
* You need also to expose your game incoming port (50007 by default) if you want to seed (`-p 50007:50007`). Also, you'll have to forward the port on your router.
* Automatic port forwarding on your router via UPnP IGD will not work unless run in `host` mode (`--net=host`).

It's more practical to run docker-compose, see Wiki Page: [DockerCompose](https://github.com/boypt/simple-game/wiki/DockerCompose)
## Source

**Requirement**
- Latest [Golang](https://golang.org/dl/) (Go 1.16+)

``` sh
$ git clone https://github.com/boypt/simple-game.git
$ cd simple-game
$ ./scripts/make_release.sh
```

# Usage

## Commandline Options
See Wiki [Command line Options](https://github.com/boypt/simple-game/wiki/Command-line-Options)

## Configuration file
See Wiki [Config File](https://github.com/boypt/simple-game/wiki/Config-File)

## Use with WEB servers (nginx/caddy)
See Wiki [Behind WebServer (reverse proxying)](https://github.com/boypt/simple-game/wiki/ReverseProxy)

# Credits 
* Credits to @jpillora for [Cloud Game](https://github.com/jpillora/cloud-game).
* Credits to @anacrolix for https://github.com/anacrolix/game
