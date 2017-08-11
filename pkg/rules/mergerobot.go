package rules

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	goimap "github.com/emersion/go-imap"
	"github.com/jim-minter/imapmagic/pkg/config"
	"github.com/jim-minter/imapmagic/pkg/imap"
)

var mergeRobotDiscard = []*regexp.Regexp{
	regexp.MustCompile("^Automatic merge from submit-queue"),
	regexp.MustCompile(`^/test all \[submit-queue is verifying that this PR is safe to merge\]`),
	regexp.MustCompile("^/lgtm cancel //PR changed after LGTM, removing LGTM"),
}
var mergeRobotSingle = []*regexp.Regexp{
	regexp.MustCompile(`^(\[APPROVALNOTIFIER\])`),
}
var mergeRobotAddr = goimap.Address{PersonalName: "OpenShift Merge Robot", MailboxName: "notifications", HostName: "github.com"}

func MergeRobot(config *config.Config, c *imap.Client) (seqset goimap.SeqSet) {
	if !config.MergeRobot.Enabled {
		return
	}

	messages := filter(c.Messages, func(message *goimap.Message) bool {
		return *message.Envelope.Sender[0] == mergeRobotAddr && githubPRRX.MatchString(message.Envelope.Subject)
	})

	m := groupBy(messages, func(message *goimap.Message) string {
		return githubPRRX.FindStringSubmatch(message.Envelope.Subject)[1]
	})

	for _, messages := range m {
		sort.Sort(sort.Reverse(byDate(messages)))

		seen := map[string]bool{}

		for _, message := range messages {
			body := messages[0].GetBody("BODY[1]").(*bytes.Buffer).String()

			if body[0] == '@' && !strings.HasPrefix(body, "@"+config.GitHub.Username+" ") {
				seqset.AddNum(message.SeqNum)
			}

			for _, rx := range mergeRobotDiscard {
				if rx.MatchString(body) {
					seqset.AddNum(message.SeqNum)
					break
				}
			}

			for _, rx := range mergeRobotSingle {
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
