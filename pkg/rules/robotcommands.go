package rules

import (
	"bytes"
	"regexp"
	"strings"

	goimap "github.com/emersion/go-imap"
	"github.com/jim-minter/imapmagic/pkg/config"
	"github.com/jim-minter/imapmagic/pkg/imap"
)

var robotCommandsRX = regexp.MustCompile(`^\s*((/retest)|(/test\s+\S+)|(\[test\]))?\s*$`)

func RobotCommands(config *config.Config, c *imap.Client) (seqset goimap.SeqSet) {
	if !config.RobotCommands.Enabled {
		return
	}

	messages := filter(c.Messages, func(message *goimap.Message) bool {
		return message.Envelope.Sender[0].MailboxName == "notifications" &&
			message.Envelope.Sender[0].HostName == "github.com"
	})

	for _, message := range messages {
		body := message.GetBody("BODY[1]").(*bytes.Buffer).String()
		matched := true
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimRight(line, "\r")

			if line == "-- " {
				break
			}

			matched = robotCommandsRX.MatchString(line)
			if !matched {
				break
			}
		}

		if matched {
			seqset.AddNum(message.SeqNum)
		}
	}

	return
}
