package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	chat "github.com/czh0526/libp2p/client/chat"
	peer "github.com/libp2p/go-libp2p-peer"
	colorable "github.com/mattn/go-colorable"

	"github.com/peterh/liner"
)

var (
	onlyWhitespace = regexp.MustCompile(`^\s*$`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)
)

const DefaultPrompt = "> "

type Config struct {
	Prompt   string
	Prompter UserPrompter
	Printer  io.Writer
}

type Console struct {
	chat     *chat.Chat
	state    CurrentState
	prompt   string
	prompter UserPrompter
	printer  io.Writer
}

func New(config Config, chat *chat.Chat) (*Console, error) {

	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}

	console := &Console{
		chat:     chat,
		prompt:   config.Prompt,
		prompter: config.Prompter,
		printer:  config.Printer,
	}
	return console, nil
}

func (c *Console) Welcome() {
	fmt.Fprintf(c.printer, "\n\nWelcome to the chat console!\n\n")
}

func (c *Console) Interactive() {
	var (
		prompt    = c.prompt
		indents   = 0
		input     = ""
		scheduler = make(chan string)
		abort     = make(chan struct{})
	)

	go func() {
		for {
			// 从 Channel 读取提示符，并等待用户在终端输入文字
			line, err := c.prompter.PromptInput(<-scheduler)
			if err != nil {
				if err == liner.ErrPromptAborted {
					// 处理 Ctrl-C
					prompt, indents, input = c.prompt, 0, ""
					scheduler <- ""
					abort <- struct{}{}
					continue
				}
				close(scheduler)
				return
			}
			// 将用户在终端输入的文字写入 Channel
			scheduler <- line
		}
	}()

	for {
		// 向 Channel 发送提示符，启动“获取用户在终端输入文字”的过程
		scheduler <- prompt
		select {
		case <-abort:
			fmt.Fprintln(c.printer, "caught interrupt, exiting")
			return
		case line, ok := <-scheduler:
			// 处理用户输入的文字
			if !ok || (indents <= 0 && exit.MatchString(line)) {
				return
			}
			if onlyWhitespace.MatchString(line) {
				continue
			}

			input += line + "\n"

			indents = countIndents(input)
			if indents <= 0 {
				prompt = c.prompt
			} else {
				prompt = strings.Repeat(".", indents*3) + " "
			}

			if indents <= 0 {
				if err := c.Evaluate(input); err != nil {
					c.Printf("error: %s \n", err)
				}
				input = ""
			}
		}
	}
}

func countIndents(input string) int {
	var (
		indents     = 0
		inString    = false
		strOpenChar = ' '   // keep track of the string open char to allow var str = "I'm ....";
		charEscaped = false // keep track if the previous char was the '\' char, allow var str = "abc\"def";
	)

	for _, c := range input {
		switch c {
		case '\\':
			// indicate next char as escaped when in string and previous char isn't escaping this backslash
			if !charEscaped && inString {
				charEscaped = true
			}
		case '\'', '"':
			if inString && !charEscaped && strOpenChar == c { // end string
				inString = false
			} else if !inString && !charEscaped { // begin string
				inString = true
				strOpenChar = c
			}
			charEscaped = false
		case '{', '(':
			if !inString { // ignore brackets when in string, allow var str = "a{"; without indenting
				indents++
			}
			charEscaped = false
		case '}', ')':
			if !inString {
				indents--
			}
			charEscaped = false
		default:
			charEscaped = false
		}
	}

	return indents
}

func (c *Console) Printf(format string, args ...interface{}) {
	fmt.Fprintf(c.printer, format, args...)
}

func (c *Console) Evaluate(statement string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(c.printer, "[native] error: %v \n", r)
		}
	}()

	if strings.HasPrefix(statement, "connect_peer:") {
		peerid := strings.TrimSpace(strings.Replace(statement, "connect_peer:", "", -1))
		pid, err := peer.IDB58Decode(peerid)
		if err != nil {
			return err
		}
		err = c.chat.ChatWithPeer(context.Background(), pid)
		if err != nil {
			c.Printf("Error: %s\n", err)
			return err
		}

		c.state.peerId = pid
		c.Printf("Current peer: %s \n", pid)
		return err

	} else if strings.HasPrefix(statement, "send_msg") {
		if !c.state.IsValidatePID() {
			c.Printf("Error: remote peer id is not set, use 'connect_peer:<...>' to set it.\n")
			return errors.New("wrong console state.")
		}

		msg := strings.TrimSpace(strings.Replace(statement, "send_msg:", "", -1))
		if err := c.chat.SendMessage(c.state.peerId, msg); err != nil {
			c.Printf("Error: %s\n", err)
			return err
		}

		return nil

	} else if strings.HasPrefix(statement, "join_group:") {
		c.Printf("*) join_group: has not be implemented.\n")
	}
	return nil
}
