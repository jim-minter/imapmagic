package rules

import (
	"bytes"
	"sort"
	"strings"

	goimap "github.com/emersion/go-imap"
	"github.com/jim-minter/imapmagic/pkg/config"
	"github.com/jim-minter/imapmagic/pkg/imap"
)

var ciRobotAddr = goimap.Address{PersonalName: "OpenShift CI Robot", MailboxName: "notifications", HostName: "github.com"}

func CIRobot(config *config.Config, c *imap.Client) (seqset goimap.SeqSet) {
	if !config.CIRobot.Enabled {
		return
	}

	messages := filter(c.Messages, func(message *goimap.Message) bool {
		return *message.Envelope.Sender[0] == ciRobotAddr && githubPRRX.MatchString(message.Envelope.Subject)
	})

	m := groupBy(messages, func(message *goimap.Message) string {
		return githubPRRX.FindStringSubmatch(message.Envelope.Subject)[1]
	})

	for _, messages := range m {
		sort.Sort(sort.Reverse(byDate(messages)))

		body := messages[0].GetBody("BODY[1]").(*bytes.Buffer).String()

		// mark newest message for discard if not for attention of config.GitHub.Username
		if body[0] == '@' && !strings.HasPrefix(body, "@"+config.GitHub.Username+": ") {
			seqset.AddNum(messages[0].SeqNum)
		}

		// mark all older messages for discard
		for _, message := range messages[1:] {
			seqset.AddNum(message.SeqNum)
		}
	}

	return
}
