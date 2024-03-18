package sentrysetup

import (
	"context"

	"github.com/jianyuan/go-sentry/v2/sentry"
	"golang.org/x/oauth2"
)

type SentrySetup struct {
	Token   string
	BaseURL string
}

func (cfg *SentrySetup) InitializeSentryClient(ctx context.Context) (*sentry.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
	tc := oauth2.NewClient(ctx, ts)
	var cl *sentry.Client
	cl, err := sentry.NewOnPremiseClient(cfg.BaseURL, tc)
	if err != nil {
		return nil, err
	}

	cl.UserAgent = "sentry-k8s-operator/0.0.1"

	return cl, nil
}
