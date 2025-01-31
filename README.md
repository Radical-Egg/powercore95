# Powercore95

A very basic dashboard to modify sandbox settings for Abiotic Factor dedicated servers. This app modifies Sandbox.ini settings and provides functionality to restart the server via the web. I am using [Pleut's docker image](https://github.com/Pleut/abiotic-factor-linux-docker) for my server.

![Demo](./doc/assets/demo.gif)

## Setup

### Environment Variables

| Variable         | Description                                                                                                                                                            |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SANDBOX_INI_PATH | The path to SandboxIniPath on your dedicated server (note this path should start with /gamefiles because ./data is mounted to /gamefiles on the powercore95 contianer) |
| CONTAINER_NAME   | The name of the abiotic-server container (should match the container_name of your abiotic server, otherwise restarts will not work)                                    |
| SERVER_NAME      | Modify the site title                                                                                                                                                  |

When making modifications to the compose file, consider the following:

- `depends_on` is not strictly needed but if you are provisioning a server for the first time
  the sandbox settings will not be created before the server starts.
- After the container is started the site will be available at http://localhost:9090 (or your server/VPS IP if you are not on the host itself)

```yaml
services:
  abiotic-server:
    container_name: abiotic-server
    image: ghcr.io/pleut/abiotic-factor-linux-docker:latest
    restart: unless-stopped
    volumes:
      - ./gamefiles:/server
      - ./data:/server/AbioticFactor/Saved
    environment:
      - MaxServerPlayers=6
      - Port=7777
      - QueryPort=27015
      - ServerPassword=password
      - SteamServerName=Linux Server
      - UsePerfThreads=true
      - NoAsyncLoadingThread=true
      - WorldSaveName=Mobo_Blob
      - AutoUpdate=true
      - AdditionalArgs=-SandboxIniPath=Config/WindowsServer/Sandbox.ini
    depends_on:
      - powercore95
    ports:
      - "0.0.0.0:7777:7777/udp"
      - "0.0.0.0:27015:27015/udp"

  powercore95:
    container_name: powercore95
    image: docker.io/radicalegg/powercore95:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/gamedata
    ports:
      - 9090:9090
    environment:
      CONTAINER_NAME: abiotic-server # abiotic-server container name for restarts
      SERVER_NAME: Mobo Blob # Sets the header on the dashboard, not related to the abiotic-server container
      SANDBOX_INI_PATH: /gamedata/Config/WindowsServer/Sandbox.ini # this default and does not need to be explicitly set
```

## Notes / Things to consider

- Mounting the docker socket (/var/run/docker.sock:/var/run/docker.sock) has some security risks/considerations to be made. I don't recommend to host this directly on the public internet. If you intend to have this world accessible, consider adding a reverse proxy + authentication (something like nginx proxy manager)
- You can not mount the docker socket and the app will still work to modify configurations, you will just need to restart the server from terminal or whatever orchestration you are using.
