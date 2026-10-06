package probe

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type multiClient []Client

func (clients multiClient) ExpireTime(ctx context.Context, domain string, host string) (time.Time, error) {
	var msgs []string
	for _, client := range clients {
		t, err := client.ExpireTime(ctx, domain, host)
		if err == nil {
			if len(msgs) > 0 {
				log.Warn().Msgf("fallback used for %s: %s", domain, strings.Join(msgs, "; "))
			}
			return t, nil
		}
		msgs = append(msgs, strings.ReplaceAll(err.Error(), "\"", "'"))
	}
	return time.Time{}, errors.New(strings.Join(msgs, "; "))
}

// NewMultiClient returns a client that wraps multiple clients.
// It returns the first success, or, if all clients fail, an error combining all failures.
func NewMultiClient(clients ...Client) Client {
	return multiClient(clients)
}
