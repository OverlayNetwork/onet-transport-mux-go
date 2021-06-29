package mux

import (
	"context"
	"testing"

	"github.com/libs4go/scf4go"
	_ "github.com/libs4go/scf4go/codec" //
	"github.com/libs4go/scf4go/reader/memory"
	"github.com/libs4go/slf4go"
	_ "github.com/libs4go/slf4go/backend/console" //
	"github.com/overlaynetwork/onet-go"
	_ "github.com/overlaynetwork/onet-transport-kcp-go" //
	"github.com/stretchr/testify/require"
)

var loggerjson = `
{
	"default":{
		"backend":"console",
		"level":"debug"
	},
	"backend":{
		"console":{
			"formatter":{
				"output": "@t @l @s @m"
			}
		}
	}
}
`

func init() {
	config := scf4go.New()

	err := config.Load(memory.New(memory.Data(loggerjson, "json")))

	if err != nil {
		panic(err)
	}

	err = slf4go.Config(config)

	if err != nil {
		panic(err)
	}

}

func TestConn(t *testing.T) {

	laddr, err := onet.NewAddr("/ip/127.0.0.1/udp/1812/kcp/mux")

	require.NoError(t, err)

	listener, err := onet.Listen(laddr)

	require.NoError(t, err)

	go func() {
		conn, err := onet.Dial(context.Background(), laddr)

		require.NoError(t, err)

		_, err = conn.Write([]byte("hello"))

		require.NoError(t, err)

		conn, err = onet.Dial(context.Background(), laddr)

		require.NoError(t, err)

		_, err = conn.Write([]byte("world"))

		require.NoError(t, err)
	}()

	conn, err := listener.Accept()

	require.NoError(t, err)

	var buff [10]byte

	n, err := conn.Read(buff[:])

	require.NoError(t, err)

	require.Equal(t, string(buff[:n]), "hello")

	conn, err = listener.Accept()

	require.NoError(t, err)

	require.NotNil(t, conn)

	n, err = conn.Read(buff[:])

	require.NoError(t, err)

	require.Equal(t, string(buff[:n]), "world")

	println(conn.LocalAddr().String(), conn.RemoteAddr().String())

	conn.Close()
}
