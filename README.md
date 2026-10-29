# Homework: Write an IRC Bot

## Goal

Write a small [IRC](https://en.wikipedia.org/wiki/Internet_Relay_Chat) bot that connects to an IRC server, joins a channel, remembers messages, and answers users in private messages.

The purpose of this assignment is to practice **TCP sockets, a text-based network protocol, parsing, and event-driven programming**. You will implement the relevant parts of the IRC protocol yourself rather than use an existing IRC library.

You may use **any programming language**. **Pascal and Go are especially encouraged and receive a grading bonus.**

## Getting started

1. Open this template repository on GitHub.
2. Select **Use this template → Create a new repository**. Do not fork the template.
3. Clone your new repository and write your bot there.
4. Commit your source code and the instructions needed to build and run it.
5. Submit the URL of **your own repository**.

## IRC server and channel

Your bot should connect to one of these servers:

| Server | Channel | Connection |
| --- | --- | --- |
| `irc.the.hell.am` | `#test` | Use the IRC port provided by the server (typically `6667` for plaintext IRC). |
| `irc.libera.chat` | `#test` | Plain TCP: port `6667`; optional TLS: port `6697`. |

Choose a distinctive nickname, preferably containing `bot`, such as `alice_bot`. The nickname must not be in use by another IRC client. Make the server, port, channel, and nickname easy to change (constants or command-line arguments are fine).

## Required behavior

### 1. Connect and remain online

The bot must:

- Establish a TCP connection to the IRC server.
- Register itself with the IRC server using `NICK` and `USER`.
- Join `#test` using `JOIN` after the server accepts its registration.
- Continue reading and processing IRC messages instead of exiting after connecting.
- Respond to server `PING` messages with `PONG` so the server does not disconnect it.

It should run until the user stops it or the connection fails. Automatic reconnection after a failure is a welcome improvement, but not required.

It should handle Ctrl+C signal, close the socket, close the file and exit gracefully.

### 2. Remember the channel's messages

The bot must record **every ordinary chat message it receives in `#test` while connected**. Store the history either:

- In memory (for example, an array, list, or queue), **or**
- In a local file.

For each message, preserve at least the **sender's nickname** and **message text**, in their original order. Timestamps are optional but encouraged.

Only messages received *after the bot joins* need to be available; the bot is not expected to retrieve older IRC history. If history is stored only in memory, it may be lost when the bot restarts.

### 3. Detect a mention and start a private conversation

If somebody mentions the bot's nickname in `#test`, the bot must answer **privately to that person**, not in the public channel.

For example, if your bot is called `alice_bot`:

```text
#test:
  bob: hello alice_bot, are you there?

Private message from alice_bot to bob:
  type 'help' for help.
```

For this assignment, a mention means the bot's nickname appears as a separate name in the channel message, such as `alice_bot` or `alice_bot:`. Do not trigger on a longer, unrelated word containing that nickname.

In IRC, starting a private conversation simply means sending a `PRIVMSG` addressed to the user's nickname. No special conversation-opening command is needed.

### 4. Answer private messages

When someone sends the bot a private message it does not recognize, it must reply:

```text
type 'help' for help.
```

When the user sends:

```text
help
```

The bot must explain the command it supports. For example:

```text
Commands:
  help       - show this help
  last N     - show the last N messages from #test (N must be a number)
```

### 5. Implement `last N`

The bot must support the following command **in private messages**:

```text
last 3
```

It must send that user the **three most recent messages** recorded from `#test`, or the channel it is in, in **chronological order** (oldest of those three first). Each reply should identify the original sender.

Example, if the bot has recorded these channel messages:

```text
<anna> first message
<bob> second message
<anna> third message
<carol> fourth message
```

Then a private request of `last 3` should produce:

```text
<bob> second message
<anna> third message
<carol> fourth message
```

**Wait at least 2 seconds between consecutive history messages sent to the user.** Sending many IRC messages quickly can trigger server flood protection and cause the bot to be disconnected.

While waiting to send history lines, your bot **must still be able to receive messages and answer `PING`**. Think about how to schedule outgoing replies without blocking the network-reading loop for a long time.

If fewer than `N` messages are available, write the other party that you have only so many messages in your memory and then return all available messages. If the channel history is empty, reply that no messages have been recorded yet.

### 6. Validate the command's argument

`N` must be a **positive integer**. In particular, if the user sends the literal command:

```text
last N
```

The bot must explain the mistake, for example:

```text
N must be a number. Try: last 5
```

Similarly, `last abc`, `last 0`, `last -2`, or `last` must not crash the bot. Respond with a helpful error. Do not treat the letter `N` as a variable or as a literal number of messages.

## IRC protocol hints

IRC is a line-oriented protocol over TCP. Commands end with **CRLF** (`\r\n`), not just `\n`.

Here are some examples of messages exchanged on the connection:

```text
# From your bot to the server (registration):
NICK alice_bot\r\n
USER alice_bot 0 * :Alice's homework bot\r\n

# After registration succeeds, join the channel:
JOIN #test\r\n

# Server sends a keepalive request:
PING :some-token\r\n

# Your bot must answer, preserving the token:
PONG :some-token\r\n

# A channel message arriving from the server:
:bob!user@host PRIVMSG #test :hello alice_bot!\r\n

# A private message arriving from the server:
:bob!user@host PRIVMSG alice_bot :last 3\r\n

# Your bot sends a private reply to bob:
PRIVMSG bob :type 'help' for help.\r\n
```

The `\r\n` sequences above represent the two terminating bytes. Do not literally send the backslash and letters `r` and `n`.

Useful details:

- The server's welcome reply (`001`) indicates successful registration; joining the channel after that is a simple approach.
- In `:bob!user@host`, the sender's nickname is `bob`.
- In `PRIVMSG #test :hello`, the target is the channel; in `PRIVMSG alice_bot :hello`, the target is the bot (a private message).
- The colon preceding the message text indicates that the remainder of the IRC line is one parameter, even if it contains spaces.
- **TCP is a byte stream:** one `read`/`recv` is not necessarily one IRC line. A line can arrive in pieces, and multiple lines can arrive together. Keep a receive buffer and separate complete `\r\n`-terminated lines.
- A socket write/send may write fewer bytes than requested. Make sure complete IRC lines are sent.

You do **not** have to implement the entire IRC protocol. Implement the parts needed for this homework.

For protocol details, see [RFC 2812](https://www.rfc-editor.org/rfc/rfc2812) (particularly `NICK`, `USER`, `JOIN`, `PING`, `PONG`, and `PRIVMSG`).

## Allowed tools and restrictions

- **Allowed:** standard Unix TCP socket operations (`socket`, `connect`, `read`, `write`, `recv`, `send`, etc.), standard language libraries, ordinary string and file I/O, and standard event mechanisms such as `select` or `poll`.
- In **Go**, the standard library (including `net`) is allowed.
- In **Free Pascal**, standard socket units (such as `Sockets`/`BaseUnix`) are allowed.
- **Not allowed:** an IRC client library, IRC bot framework, or a library that performs IRC registration, message parsing, or commands on your behalf.
- Do not implement the assignment by controlling an existing IRC client or using `nc` as a subprocess. Your program must manage its own connection.
- TLS is optional for this exercise; if you use it, the same rule applies: **no IRC-specific library**.

## How to test your bot

Use a normal IRC client (for example, WeeChat, irssi, or Pidgin) as the human user. Check the following:

1. The bot connects, joins `#test`, and stays there.
2. It survives the server's `PING` requests.
3. You send several messages to `#test`; they are recorded in order.
4. You mention the bot's nickname in the channel; it responds to you **privately**, not publicly.
5. You send the bot a private `hello`; it replies `type 'help' for help.`
6. You send `help`; it describes `last N`.
7. You send `last 3`; you receive the last three channel messages, in order, spaced at least **2 seconds apart**.
8. You send `last N`, `last abc`, `last 0`, and `last`; it gives useful errors and continues running.
9. You request more messages than have been recorded; the bot returns only the messages it has.
10. You send new channel messages while history is being replayed; the bot continues processing incoming traffic.

**Do not flood a public network while testing.** Remember that channel history may contain other people's messages. Do not publish chat logs in your GitHub repository.

## What to submit

Your GitHub repository must contain:

- Your bot's **source code**.
- A short description of your implementation and its chosen storage method.
- **Build and run instructions**, including any compiler/runtime dependencies.
- The server, port, channel, and nickname you used during testing (no passwords or credentials).

Your program should work on a Unix-like system, such as Linux. Do not commit compiled executables, secrets, or captured conversations.

## Grading

| Requirement | Points |
| --- | ---: |
| Connects, registers, joins, and remains running | 15 |
| Correct `PING` / `PONG` handling | 15 |
| Records channel messages in order | 15 |
| Recognizes mentions and starts private conversations | 15 |
| Handles private `help` and invalid commands/arguments | 15 |
| Correct `last N` behavior, including 2-second spacing | 20 |
| Readable source code and build/run instructions | 5 |
| **Total** | **100** |

**Language bonus:** **+10 points** for an implementation in **Pascal or Go**, provided it satisfies the same functional requirements. All other languages are welcome.

The bot must be your own implementation. Be prepared to explain how your code processes the IRC stream, stores channel messages, and avoids blocking while sending history.
