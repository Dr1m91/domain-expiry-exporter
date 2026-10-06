package probe

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestMulti_FallbackWarning(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = orig })

	ctx := context.Background()

	_, err := NewMultiClient(clifail(0), clisuccess(time.Now())).ExpireTime(ctx, "a", "")
	require.NoError(t, err)
	require.Contains(t, buf.String(), "fallback used for a")

	buf.Reset()
	_, err = NewMultiClient(clisuccess(time.Now()), clifail(0)).ExpireTime(ctx, "a", "")
	require.NoError(t, err)
	require.Empty(t, buf.String())

	buf.Reset()
	_, err = NewMultiClient(clifail(0), clifail(0)).ExpireTime(ctx, "a", "")
	require.Error(t, err)
	require.Empty(t, buf.String())
}
