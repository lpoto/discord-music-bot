# [Discord Music Bot](https://github.com/lpoto/discord-music-bot)

A simple, easy to use discord bot, intended for playing youtube songs.

It sends a single message in the discord server representing a music queue
and all it's commands are available as buttons.

**_NOTE_** this bot is under development, so there may be some bugs.
Report any issues [here](https://github.com/lpoto/discord-music-bot/issues).

## Usage

- Use `/music` to start a new muisc queue in the channel.

  - Only a single queue may be active in a single discord server.

- Add songs by clicking the `Add` button.

  - Multiple songs may be added at a time, by typing them each in their own line.
  - Either the name or the url to a Youtube song may be typed to add the desired song.

- The queue displays up to 10 songs at once, navigate the queue with `<` and `>` buttons.

- To skip a song, press `>>`, to go back to the previous song, click `<<`.

- To pause the song, click the `||` button.

- To replay the currently playing song from the start, press `↺`

- To enable loop, click on `Loop`.

  - When loop is enabled, songs are not removed from the queue but rather pushed to the back of the queue.

- Bot will leave the channel after being alone for 2 minutes.

  - The queue won't be deleted, click on `Join` and the bot will start playing again.

- Bot will disconnect if the queue message is deleted.

- Stop the music with `/stop`. (This is only temporary)

See [develop](https://github.com/lpoto/discord-music-bot/blob/main/doc/develop.md)
