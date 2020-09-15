package mux

import (
	"context"
	"testing"

	"github.com/overlaynetwork/onet-go"
	_ "github.com/overlaynetwork/onet-transport-kcp-go" //
	"github.com/stretchr/testify/require"
)

func TestConn(t *testing.T) {

	laddr, err := onet.NewAddr("/ip/127.0.0.1/udp/1812/kcp/mux")

	require.NoError(t, err)

	listener, err := onet.Listen(laddr)

	require.NoError(t, err)

	go func() {
		conn, err := onet.Dial(context.Background(), laddr)

		require.NoError(t, err)

		_, err = conn.Write([]byte("hello world"))

		require.NoError(t, err)

		conn, err = onet.Dial(context.Background(), laddr)

		require.NoError(t, err)

		_, err = conn.Write([]byte("hello world"))

		require.NoError(t, err)
	}()

	_, err = listener.Accept()

	require.NoError(t, err)

	conn, err := listener.Accept()

	require.NoError(t, err)

	require.NotNil(t, conn)

	transport, ok := conn.ONet().MuxTransports[0].(*muxTransport)

	require.True(t, ok)

	require.Equal(t, len(transport.out), 1)

	require.Equal(t, len(transport.accpetChan), 1)
}
