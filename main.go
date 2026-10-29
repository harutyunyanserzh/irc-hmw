// ircbot - a small IRC bot for the networks26 irc-hmw homework.
// It speaks raw IRC over a TCP socket (no IRC libraries).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
)

// Message is one chat line recorded from the channel.
type Message struct {
	Time time.Time
	Nick string
	Text string
}

type Bot struct {
	conn    net.Conn
	nick    string
	channel string

	sendMu sync.Mutex // only one goroutine writes to the socket at a time

	histMu  sync.Mutex // protects history and logFile
	history []Message
	logFile *os.File
}

func main() {
	server := flag.String("server", "the.hell.am", "IRC server host")
	port := flag.Int("port", 6667, "IRC server port")
	channel := flag.String("channel", "#test", "channel to join")
	nick := flag.String("nick", "serzh_bot", "bot nickname")
	logPath := flag.String("log", "history.log", "file to append channel messages to (empty = memory only)")
	flag.Parse()

	addr := net.JoinHostPort(*server, strconv.Itoa(*port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("connect %s: %v", addr, err)
	}
	log.Printf("connected to %s", addr)

	b := &Bot{conn: conn, nick: *nick, channel: *channel}
	if *logPath != "" {
		f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("open %s: %v", *logPath, err)
		}
		b.logFile = f
	}

	// Ctrl+C / SIGTERM: say goodbye, close socket and file, exit.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-sigs
		log.Printf("got %v, shutting down", s)
		b.shutdown(0)
	}()

	// Registration.
	b.send("NICK " + b.nick)
	b.send("USER " + b.nick + " 0 * :networks26 homework bot")

	// Main loop: bufio.Reader keeps a receive buffer, so lines that arrive
	// in pieces (or several at once) are split correctly on '\n'.
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("connection closed: %v", err)
			b.shutdown(1)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		log.Printf("<< %s", line)
		b.handle(line)
	}
}

func (b *Bot) shutdown(code int) {
	b.send("QUIT :bye")
	b.conn.Close()
	b.histMu.Lock()
	if b.logFile != nil {
		b.logFile.Close()
		b.logFile = nil
	}
	b.histMu.Unlock()
	os.Exit(code)
}

// send writes one complete IRC line terminated by CRLF.
// net.Conn.Write keeps writing until all bytes are sent or an error occurs.
func (b *Bot) send(line string) {
	b.sendMu.Lock()
	defer b.sendMu.Unlock()
	if _, err := b.conn.Write([]byte(line + "\r\n")); err != nil {
		log.Printf("send error: %v", err)
		return
	}
	log.Printf(">> %s", line)
}

func (b *Bot) privmsg(to, text string) {
	b.send("PRIVMSG " + to + " :" + text)
}

// parse splits ":prefix COMMAND p1 p2 :trailing text" into its parts.
func parse(line string) (prefix, cmd string, params []string) {
	if strings.HasPrefix(line, ":") {
		i := strings.IndexByte(line, ' ')
		if i < 0 {
			return
		}
		prefix, line = line[1:i], line[i+1:]
	}
	trailing, hasTrailing := "", false
	if i := strings.Index(line, " :"); i >= 0 {
		trailing, line, hasTrailing = line[i+2:], line[:i], true
	} else if strings.HasPrefix(line, ":") {
		trailing, line, hasTrailing = line[1:], "", true
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	cmd, params = strings.ToUpper(fields[0]), fields[1:]
	if hasTrailing {
		params = append(params, trailing)
	}
	return
}

// nickOf turns "bob!user@host" into "bob".
func nickOf(prefix string) string {
	if i := strings.IndexByte(prefix, '!'); i >= 0 {
		return prefix[:i]
	}
	return prefix
}

func (b *Bot) handle(line string) {
	prefix, cmd, params := parse(line)
	switch cmd {
	case "PING":
		token := ""
		if len(params) > 0 {
			token = params[len(params)-1]
		}
		b.send("PONG :" + token)
	case "001": // welcome = registration accepted
		b.send("JOIN " + b.channel)
	case "433": // nickname in use: try another one (server limit is 9 chars)
		if len(b.nick) >= 9 {
			b.nick = b.nick[:8]
		}
		b.nick += "_"
		b.send("NICK " + b.nick)
	case "PRIVMSG":
		if len(params) < 2 {
			return
		}
		b.onPrivmsg(nickOf(prefix), params[0], params[1])
	}
}

func (b *Bot) onPrivmsg(from, target, text string) {
	if from == "" || strings.HasPrefix(text, "\x01") { // ignore CTCP
		return
	}
	switch {
	case strings.EqualFold(target, b.channel):
		b.record(from, text)
		if mentions(text, b.nick) {
			b.privmsg(from, "type 'help' for help.")
		}
	case strings.EqualFold(target, b.nick):
		b.command(from, text)
	}
}

func isNickChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_-[]\\`^{}|", r)
}

// mentions is true only if nick appears as a separate word
// ("serzh_bot" or "serzh_bot:"), not inside a longer word.
func mentions(text, nick string) bool {
	words := strings.FieldsFunc(text, func(r rune) bool { return !isNickChar(r) })
	for _, w := range words {
		if strings.EqualFold(w, nick) {
			return true
		}
	}
	return false
}

func (b *Bot) record(nick, text string) {
	m := Message{Time: time.Now(), Nick: nick, Text: text}
	b.histMu.Lock()
	defer b.histMu.Unlock()
	b.history = append(b.history, m)
	if b.logFile != nil {
		fmt.Fprintf(b.logFile, "%s <%s> %s\n", m.Time.Format("2006-01-02 15:04:05"), m.Nick, m.Text)
	}
}

func (b *Bot) command(from, text string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		b.privmsg(from, "type 'help' for help.")
		return
	}
	switch strings.ToLower(fields[0]) {
	case "help":
		b.privmsg(from, "Commands:")
		b.privmsg(from, "help - show this help")
		b.privmsg(from, "last N - show the last N messages from "+b.channel+" (N must be a positive number, e.g. last 5)")
	case "last":
		if len(fields) != 2 {
			b.privmsg(from, "Usage: last N, where N is a positive number. Try: last 5")
			return
		}
		arg := fields[1]
		if arg == "N" || arg == "n" {
			b.privmsg(from, "N must be a number. Try: last 5")
			return
		}
		n, err := strconv.Atoi(arg)
		if err != nil || n <= 0 {
			b.privmsg(from, "N must be a positive whole number, not '"+arg+"'. Try: last 5")
			return
		}
		// Run in its own goroutine so the 2-second pauses never block
		// the main loop (which keeps reading messages and answering PING).
		go b.sendHistory(from, n)
	default:
		b.privmsg(from, "type 'help' for help.")
	}
}

func (b *Bot) sendHistory(to string, n int) {
	b.histMu.Lock()
	total := len(b.history)
	start := total - n
	if start < 0 {
		start = 0
	}
	msgs := append([]Message(nil), b.history[start:]...) // copy
	b.histMu.Unlock()

	if total == 0 {
		b.privmsg(to, "No messages have been recorded yet.")
		return
	}
	if total < n {
		b.privmsg(to, fmt.Sprintf("I have only %d message(s) in my memory. Here they are:", total))
		time.Sleep(2 * time.Second)
	}
	for i, m := range msgs {
		if i > 0 {
			time.Sleep(2 * time.Second) // flood protection
		}
		b.privmsg(to, fmt.Sprintf("<%s> %s", m.Nick, m.Text))
	}
}
