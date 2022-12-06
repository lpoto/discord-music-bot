# [Discord Music Bot](https://github.com/lpoto/discord-music-bot)

A simple, easy to use discord bot, intended for playing youtube songs.

It sends a single message in the discord server representing a music queue
and all it's commands are available as buttons.

**_NOTE_** this bot is under development, so there may be some bugs.
Report any issues [here](https://github.com/lpoto/discord-music-bot/issues).

## Usage

- Use `/music` to start a new muisc queue in the channel.

  > Only a single queue may be active in a single discord server.

- `Add` button opens a modal through which songs may be added.

  > Multiple songs may be added at a time, by typing them each in their own line.
  > Either the name or the url to a Youtube song may be typed to add the desired song.

- `<`, `>` buttons allow you to navigate through the displayed songs.

- `Loop` button enables loop.

  > When loop is enabled, songs are not removed from the queue but rather pushed to the back of the queue.

- `>>` button skips the currently playing song.

- `<<` button starts playing the previous song.

  > Previous songs are deleted after a few hours.
  > If _loop_ is enabled, the button will start playing the last song in the queue.

- `||` button pauses the currently playing song.

- `↺` button replays the currently playing song.

- Bot will leave the channel after being alone for 2 minutes.

  > The queue won't be deleted, click on `Join` and the bot will start playing again.

- Bot will disconnect if the queue message is deleted.

- Stop the music with `/stop`. (This is only temporary)

To help developing the bot see [develop](https://github.com/lpoto/discord-music-bot/blob/main/doc/develop.md)
