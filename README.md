# IRC bot (networks26 irc-hmw)

A small IRC bot written in **Go** using only the standard library (`net`, `bufio`), with no IRC libraries.

## What it does
- Connects over TCP, registers with NICK/USER, joins the channel after the 001 welcome.
- Answers server PING with PONG (same token) so it stays online.
- Records every channel PRIVMSG (sender, text, time) in order.
- If its nickname is mentioned as a separate word in the channel, it replies privately: `type 'help' for help.`
- Private commands:
  - `help` lists the commands
  - `last N` sends the last N channel messages, oldest first, 2 seconds apart
  - invalid input (`last N`, `last abc`, `last 0`, `last`) gets a helpful error; anything else gets `type 'help' for help.`
- Ctrl+C (SIGINT/SIGTERM): sends QUIT, closes the socket and the log file, exits.

## Implementation notes
- Reading: `bufio.Reader.ReadString('\n')` keeps a receive buffer, so partial or merged TCP reads become complete IRC lines.
- Writing: every line goes through one mutex-protected `send()` that appends `\r\n`; `net.Conn.Write` writes the whole buffer or returns an error.
- Non-blocking history: `last N` runs in its own goroutine, so the 2-second pauses never block the main loop, which keeps reading messages and answering PING.
- Storage: in memory (a slice of {time, nick, text}), also appended to a local `history.log` file (git-ignored, not committed).

## Build and run
Requires Go 1.18 or newer.

    go build -o ircbot .
    ./ircbot
    ./ircbot -server irc.libera.chat -port 6667 -channel '#test' -nick serzh_bot
    ./ircbot -log ""      # memory only, no log file

## Tested with
- Server: the.hell.am (irc.the.hell.am), port 6667, plain TCP
- Channel: #test
- Nickname: serzh_bot
