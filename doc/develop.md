# Developing the music bot

## Contents

1. [Prerequisites](#prerequisites)
2. [Running the bot inside a docker container](#running-the-bot-inside-a-docker-container)
3. [Running the bot wihtout docker](#running-the-bot-without-docker)
4. [Building the docker image](#building-the-image)
5. [Creating a Discord Bot Token](#creating-a-discord-bot-token)
6. [Adding the bot to a Discord Server](#add-the-bot-to-your-discord-server)

## Prerequisites

1. Create the file `./src/config.yaml` then copy and modify the contents
   from [./conf/config.example.yaml](./conf/config.example.yaml).
2. Make sure the datastore values match an existing postgresql instance.
3. Update [help.txt](./conf/help.txt) if necessary.

## Running the bot inside a docker container

```bash
docker-compose -f .dockerenv/docker-compose.yaml up
```

## Running the bot without docker

1. Run the postgres container (or set it up locally without the docker):

```bash
cd ./src

go run .
```

## Building the image

Running:

```bash
.dockerenv/build
```

builds the bot image and pushes it to [docker hub](https://hub.docker.com/).
To use the built image, replace the `build:` section in [docker-compose.yaml](./.dockerenv/docker-compose.yaml)
with `image: <built-image-reference>`.

## Creating a discord bot token

1. Visit [discord developer portal](https://discord.com/developers) and log in to your discord account.

2. Under `Applications` click on `New application` and name your discord bot.

3. Under `Bot` click on `Add bot` and then:

- under `Privileged Gateway Intents` check `PRESENCE INTENT`, `SERVER MEMBERS INTENT` and `MESSAGE CONTENT INTENT`,
- save `TOKEN` so it may be used in the config

## Add the bot to your discord server

- Under `OAuth2/URL Generator` under `SCOPES` select:
- bot,
- applications.commands
- Under `BOT PERMISSIONS` select:
- Send Messages,
- Connect,
- Speak
- Copy `GENERATED URL` and paste it into the browser.
