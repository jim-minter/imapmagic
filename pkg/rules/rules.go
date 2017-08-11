package rules

import (
	"regexp"

	goimap "github.com/emersion/go-imap"
	"github.com/jim-minter/imapmagic/pkg/config"
	"github.com/jim-minter/imapmagic/pkg/imap"
)

type Rule func(*config.Config, *imap.Client) goimap.SeqSet

var githubPRRX = regexp.MustCompile(` \(#(\d+)\)$`)

func filter(messages []*goimap.Message, f func(*goimap.Message) bool) []*goimap.Message {
	var m []*goimap.Message

	for _, message := range messages {
		if f(message) {
			m = append(m, message)
		}
	}

	return m
}

func groupBy(messages []*goimap.Message, f func(*goimap.Message) string) map[string][]*goimap.Message {
	m := map[string][]*goimap.Message{}

	for _, message := range messages {
		group := f(message)
		m[group] = append(m[group], message)
	}

	return m
}

type byDate []*goimap.Message

func (s byDate) Len() int           { return len(s) }
func (s byDate) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byDate) Less(i, j int) bool { return s[i].Envelope.Date.Before(s[j].Envelope.Date) }
