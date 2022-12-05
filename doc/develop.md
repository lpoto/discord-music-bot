# Developing the music bot

## Contents

1. [Prerequisites](#prerequisites)
2. [Running the bot](#running-the-bot)
3. [Running tests](#running-tests)
4. [Creating a Discord Bot Token](#creating-a-discord-bot-token)
5. [Adding the bot to a Discord Server](#add-the-bot-to-your-discord-server)

## Prerequisites

1. Create the file `./config.yaml` then copy and modify the contents
   from [./config.example.yaml](./config.example.yaml).
2. Create a [discord token] and add it to `./config.yaml`.
3. Make sure the `Datastore/Postgres` values in `./config.yaml`
   match a running postgresql instance.

## Running the bot

```bash
cd ./src
go run .
```

## Running tests

Tests are run with github's CI, but to run them locally:

```bash
docker-compose -f .github/dockerenv/docker-compose.test.yaml up -d

docker-compose -f .github/dockerenv/docker-compose.test.yaml exec bot bash
go test ./... -p 1
```

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
