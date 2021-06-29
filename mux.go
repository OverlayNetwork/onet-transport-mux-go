package mux

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"github.com/inconshreveable/muxado"
	"github.com/libs4go/errors"
	"github.com/libs4go/slf4go"
	"github.com/overlaynetwork/onet-go"
)

type muxSession struct {
	conn    onet.Conn
	session muxado.Session
	addr    *onet.Addr
	counter int64
	network *onet.OverlayNetwork
}

func newSession(network *onet.OverlayNetwork, addr *onet.Addr, next onet.Next, isServer bool) (*muxSession, error) {

	conn, err := next()

	if err != nil {
		return nil, err
	}

	var session muxado.Session

	if isServer {
		session = muxado.Server(conn, nil)
	} else {
		session = muxado.Client(conn, nil)
	}

	return &muxSession{
		session: session,
		conn:    conn,
		addr:    addr,
		network: network,
	}, nil
}

func (session *muxSession) Close() error {
	return nil
}

func (session *muxSession) Accept() (*muxConn, error) {
	conn, err := session.session.Accept()

	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&session.counter, 1)

	return newMuxConn(conn, session)
}

func (session *muxSession) Connect() (*muxConn, error) {
	conn, err := session.session.Open()

	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&session.counter, 1)

	return newMuxConn(conn, session)
}

type muxConn struct {
	net.Conn
	session *muxSession
	laddr   *onet.Addr
	raddr   *onet.Addr
}

func newMuxConn(conn net.Conn, session *muxSession) (*muxConn, error) {

	_, relativeAddr, err := session.addr.ResolveNetAddr()

	if err != nil {
		return nil, err
	}

	lNetaddr, _, err := session.conn.LocalAddr().ResolveNetAddr()

	if err != nil {
		return nil, err
	}

	laddr, err := onet.FromNetAddr(lNetaddr)

	if err != nil {
		return nil, err
	}

	rNetaddr, _, err := session.conn.RemoteAddr().ResolveNetAddr()

	if err != nil {
		return nil, err
	}

	raddr, err := onet.FromNetAddr(rNetaddr)

	if err != nil {
		return nil, err
	}

	laddr = laddr.Join(relativeAddr.SubAddrs()...)

	raddr = raddr.Join(relativeAddr.SubAddrs()...)

	return &muxConn{
		Conn:    conn,
		session: session,
		laddr:   laddr,
		raddr:   raddr,
	}, nil
}

func (conn *muxConn) LocalAddr() *onet.Addr {

	return conn.laddr
}

func (conn *muxConn) RemoteAddr() *onet.Addr {
	return conn.raddr
}

func (conn *muxConn) ONet() *onet.OverlayNetwork {
	return conn.session.network
}

func (conn *muxConn) Close() error {
	conn.Conn.Close()

	atomic.AddInt64(&conn.session.counter, -1)

	return nil
}

type muxTransport struct {
	slf4go.Logger
	sync.RWMutex
	out map[string]*muxSession
	in  map[string]*muxSession
}

func newMuxTransport() *muxTransport {
	return &muxTransport{
		Logger: slf4go.Get("mux"),
		out:    make(map[string]*muxSession),
		in:     make(map[string]*muxSession),
	}
}

func (transport *muxTransport) String() string {
	return transport.Protocol()
}

func (transport *muxTransport) Protocol() string {
	return "mux"
}

func (transport *muxTransport) search(sessions map[string]*muxSession, network *onet.OverlayNetwork, addr *onet.Addr, next onet.Next, isServer bool) (*muxSession, error) {

	transport.RLock()
	session, ok := sessions[addr.String()]
	transport.RUnlock()

	if !ok {
		var err error
		session, err = newSession(network, addr, next, isServer)

		if err != nil {
			return nil, err
		}

		transport.Lock()
		sessions[addr.String()] = session
		transport.Unlock()
	}

	return session, nil
}

func (transport *muxTransport) close(sessions map[string]*muxSession, addr *onet.Addr) {
	transport.Lock()
	defer transport.Unlock()

	session, ok := sessions[addr.String()]

	if ok {
		delete(sessions, addr.String())

		if err := session.Close(); err != nil {
			transport.E("close session error {@e}", err)
		}
	}
}

func (transport *muxTransport) Client(ctx context.Context, network *onet.OverlayNetwork, addr *onet.Addr, next onet.Next) (onet.Conn, error) {

	session, err := transport.search(transport.out, network, addr, next, false)

	if err != nil {
		return nil, err
	}

	conn, err := session.Connect()

	if err != nil {
		transport.close(transport.out, addr)
		return nil, errors.Wrap(err, "mux session %s open stream error", addr)
	}

	return conn, nil

}

func (transport *muxTransport) Server(ctx context.Context, network *onet.OverlayNetwork, addr *onet.Addr, next onet.Next) (onet.Conn, error) {

	session, err := transport.search(transport.in, network, addr, next, true)

	if err != nil {
		return nil, err
	}

	conn, err := session.Accept()

	if err != nil {
		transport.close(transport.out, addr)
		return nil, errors.Wrap(err, "mux session %s accept error", addr)
	}

	return conn, nil
}

func (transport *muxTransport) Close(network *onet.OverlayNetwork, addr *onet.Addr, next onet.NextClose) error {
	transport.Lock()
	session, ok := transport.in[addr.String()]

	if ok {
		delete(transport.in, addr.String())
		session.Close()
	}

	transport.Unlock()

	return next()

}

var protocol = &onet.Protocol{Name: "mux"}

func init() {

	if err := onet.RegisterProtocol(protocol); err != nil {
		panic(err)
	}

	if err := onet.RegisterTransport(newMuxTransport()); err != nil {
		panic(err)
	}
}
