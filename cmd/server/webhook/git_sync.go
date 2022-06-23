package webhook

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/lunarway/release-manager/internal/git"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func GitSyncHandler(gitSvc *git.Service, githubWebhookSecret string) func([]byte) error {
	isBranchPush := func(ref string) bool {
		return strings.HasPrefix(ref, "refs/heads/")
	}

	return func(msgBody []byte) error {
		// parse msgBody as github.PushPayload
		var payload github.PushPayload
		err := json.Unmarshal(msgBody, &payload)
		if err != nil {
			return err
		}

		// Only synch on branch pushes
		if !isBranchPush(payload.Ref) {
			return nil
		}

		err = gitSvc.SyncMaster(context.Background())
		if err != nil {
			return err
		}

		return nil
	}
}
