package imap

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap-move"
	"github.com/emersion/go-imap/client"
	"github.com/jim-minter/imapmagic/pkg/config"
)

type Client struct {
	client     *client.Client
	moveClient *move.Client
	idleClient *idle.Client
	updates    chan interface{}
	Messages   []*imap.Message
}

func Connect(config *config.Config, debug bool) (c *Client, err error) {
	c = &Client{}

	c.client, err = client.DialTLS(config.Imap.Server, nil)
	if err != nil {
		return
	}

	if debug {
		c.client.SetDebug(os.Stderr)
	} else {
		c.client.ErrorLog = discard{}
	}

	c.updates = make(chan interface{})
	c.client.Updates = c.updates

	err = c.client.Login(config.Imap.Username, config.Imap.Password)
	if err != nil {
		return
	}

	c.moveClient = move.NewClient(c.client)
	supportMove, err := c.moveClient.SupportMove()
	if err != nil {
		return
	}
	if !supportMove {
		err = fmt.Errorf("%s not supported", move.Capability)
		return
	}

	c.idleClient = idle.NewClient(c.client)
	supportIdle, err := c.idleClient.SupportIdle()
	if err != nil {
		return
	}
	if !supportIdle {
		err = fmt.Errorf("%s not supported", idle.Capability)
		return
	}

	return
}

func (c *Client) Fetch(seqset *imap.SeqSet, items []string) (messages []*imap.Message, err error) {
	done := make(chan struct{})
	ch := make(chan *imap.Message)

	go func() {
		err = c.client.Fetch(seqset, items, ch)
		close(done)
	}()

	for msg := range ch {
		messages = append(messages, msg)
	}
	<-done
	return
}

func (c *Client) Idle(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 29*time.Minute)
	defer cancel()

	done := make(chan struct{})
	var updates []interface{}

	go func() {
		err = c.idleClient.Idle(ctx.Done())
		close(done)
	}()

	for {
		select {
		case u := <-c.updates:
			updates = append(updates, u)
			if _, ok := u.(*client.MailboxUpdate); ok {
				cancel()
			}
		case <-done:
			if err != nil {
				return
			}
			return c.handleUpdates(updates)
		}
	}
}

func (c *Client) Move(seqset *imap.SeqSet, dest string) (err error) {
	done := make(chan struct{})
	var updates []interface{}

	go func() {
		err = c.moveClient.Move(seqset, dest)
		close(done)
	}()

	for {
		select {
		case u := <-c.updates:
			updates = append(updates, u)
		case <-done:
			if err != nil {
				return
			}
			return c.handleUpdates(updates)
		}
	}
}

func (c *Client) Logout() (err error) {
	done := make(chan struct{})

	go func() {
		err = c.client.Logout()
		close(done)
	}()

	for {
		select {
		case <-c.updates:
		case <-done:
			return
		}
	}
}

func (c *Client) Select(name string, readOnly bool) (err error) {
	done := make(chan struct{})

	go func() {
		_, err = c.client.Select(name, readOnly)
		close(done)
	}()

out:
	for {
		select {
		case <-c.updates:
		case <-done:
			break out
		}
	}

	if err != nil {
		return err
	}

	c.Messages, err = c.Fetch(&imap.SeqSet{Set: []imap.Seq{{Start: 1, Stop: 0}}}, []string{imap.EnvelopeMsgAttr, "BODY.PEEK[1]"})
	return err
}

func (c *Client) handleUpdates(updates []interface{}) error {
	for _, u := range updates {
		switch u := u.(type) {
		case *client.ExpungeUpdate:
			c.Messages = append(c.Messages[:u.SeqNum-1], c.Messages[u.SeqNum:]...)
		case *client.MailboxUpdate:
			if u.Mailbox.Messages < uint32(len(c.Messages)) {
				panic(fmt.Sprintf("underflow %d %d", u.Mailbox.Messages, len(c.Messages)))
			}
			if u.Mailbox.Messages > uint32(len(c.Messages)) {
				newmessages, err := c.Fetch(&imap.SeqSet{Set: []imap.Seq{{Start: uint32(len(c.Messages) + 1), Stop: u.Mailbox.Messages}}}, []string{imap.EnvelopeMsgAttr, "BODY.PEEK[1]"})
				if err != nil {
					return err
				}
				c.Messages = append(c.Messages, newmessages...)
			}
		default:
			panic(u)
		}
	}

	return nil
}

type discard struct{}

func (discard) Printf(string, ...interface{}) {}
func (discard) Println(...interface{})        {}

var _ imap.Logger = discard{}
