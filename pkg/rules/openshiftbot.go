package rules

import (
	"bytes"
	"regexp"
	"sort"

	goimap "github.com/emersion/go-imap"
	"github.com/jim-minter/imapmagic/pkg/config"
	"github.com/jim-minter/imapmagic/pkg/imap"
)

var openshiftBotDiscard = []*regexp.Regexp{
	regexp.MustCompile(`^[^\n]+ Merge Results: ((Evaluating)|(Running))`),
	regexp.MustCompile(`^Evaluated for [^\n]+ ((test)|(merge)) up to`),
	regexp.MustCompile(`^[^ ]+ Evaluating for testing`),
	regexp.MustCompile(`^[^ ]+ Running`),
}
var openshiftBotSingle = []*regexp.Regexp{
	regexp.MustCompile(`^([^\n]+ Merge Results): ((SUCCESS)|(FAILURE))`),
	regexp.MustCompile(`^([^ ]+) ((SUCCESS)|(FAILURE))`),
}
var openshiftBotAddr = goimap.Address{PersonalName: "OpenShift Bot", MailboxName: "notifications", HostName: "github.com"}

func OpenshiftBot(config *config.Config, c *imap.Client) (seqset goimap.SeqSet) {
	if !config.OpenshiftBot.Enabled {
		return
	}

	messages := filter(c.Messages, func(message *goimap.Message) bool {
		return *message.Envelope.Sender[0] == openshiftBotAddr && githubPRRX.MatchString(message.Envelope.Subject)
	})

	m := groupBy(messages, func(message *goimap.Message) string {
		return githubPRRX.FindStringSubmatch(message.Envelope.Subject)[1]
	})

	for _, messages := range m {
		sort.Sort(sort.Reverse(byDate(messages)))

		seen := map[string]bool{}

		for _, message := range messages {
			body := message.GetBody("BODY[1]").(*bytes.Buffer).String()

			for _, rx := range openshiftBotDiscard {
				if rx.MatchString(body) {
					seqset.AddNum(message.SeqNum)
					break
				}
			}

			for _, rx := range openshiftBotSingle {
				match := rx.FindStringSubmatch(body)
				if match != nil {
					if seen[match[1]] {
						seqset.AddNum(message.SeqNum)
					}
					seen[match[1]] = true
				}
			}
		}
	}

	return
}
