package mux

import (
	"context"
	"sync"

	"github.com/inconshreveable/muxado"
	"github.com/libs4go/errors"
	"github.com/overlaynetwork/onet-go"
)

type muxConn struct {
	conn onet.Conn
	err  error
}

type muxSession struct {
	conn    onet.Conn
	session muxado.Session
}

type muxTransport struct {
	sync.RWMutex
	out        map[string]*muxSession
	accpetChan map[string]chan *muxConn
}

type muxListener struct {
	transport  *muxTransport
	laddr      *onet.Addr
	accpetChan chan *muxConn
}

func (listener *muxListener) Accept() (onet.Conn, error) {
	conn := <-listener.accpetChan
	return conn.conn, conn.err
}

func (listener *muxListener) Close() error {
	return listener.transport.close(listener.laddr)
}

func (listener *muxListener) Addr() *onet.Addr {
	return listener.laddr
}

func newMuxTransport() *muxTransport {
	return &muxTransport{
		out:        make(map[string]*muxSession),
		accpetChan: make(map[string]chan *muxConn),
	}
}

func (transport *muxTransport) close(laddr *onet.Addr) error {
	transport.Lock()
	defer transport.Unlock()

	acceptChan, ok := transport.accpetChan[laddr.String()]

	if ok {
		close(acceptChan)
	}

	return nil
}

func (transport *muxTransport) String() string {
	return transport.Protocol()
}

func (transport *muxTransport) Protocol() string {
	return "mux"
}

func (transport *muxTransport) Listen(network *onet.OverlayNetwork, chainOffset int) (onet.Listener, error) {

	transport.Lock()
	defer transport.Unlock()

	acceptChan, ok := transport.accpetChan[network.Addr.String()]

	if ok {
		return nil, errors.Wrap(onet.ErrAddr, "laddr %s already bind", network.Addr)
	}

	acceptChan = make(chan *muxConn)
	transport.accpetChan[network.Addr.String()] = acceptChan

	return &muxListener{
		transport:  transport,
		laddr:      network.Addr,
		accpetChan: acceptChan,
	}, nil
}

func (transport *muxTransport) Dial(ctx context.Context, network *onet.OverlayNetwork, chainOffset int) (onet.Conn, error) {

	transport.RLock()
	session, ok := transport.out[network.Addr.String()]
	transport.RUnlock()

	if !ok {
		return nil, errors.Wrap(onet.ErrMuxNotFound, "mux %s session %s not found", network.MuxAddrs[chainOffset], network.Addr)
	}

	conn, err := session.session.Open()

	if err != nil {
		return nil, errors.Wrap(onet.ErrMuxNotFound, "mux %s session %s open stream error", network.MuxAddrs[chainOffset], network.Addr)
	}

	return onet.ToOnetConnWithAddr(conn, network, session.conn.LocalAddr(), session.conn.RemoteAddr())
}

func (transport *muxTransport) Client(network *onet.OverlayNetwork, conn onet.Conn, chainOffset int) (onet.Conn, error) {

	transport.Lock()
	session, ok := transport.out[network.Addr.String()]

	if ok {
		session.session.Close()
	}

	session = &muxSession{
		session: muxado.Client(conn, nil),
		conn:    conn,
	}

	transport.out[network.Addr.String()] = session

	transport.Unlock()

	sessionConn, err := session.session.Open()

	if err != nil {
		return nil, errors.Wrap(onet.ErrMuxNotFound, "mux %s session %s open stream error", network.MuxAddrs[chainOffset], network.Addr)
	}

	result, err := onet.ToOnetConnWithAddr(sessionConn, network, conn.LocalAddr(), conn.RemoteAddr())

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (transport *muxTransport) doAccept(network *onet.OverlayNetwork, underlying onet.Conn, session muxado.Session, acceptChain chan *muxConn) {

	defer recover()

	for {
		conn, err := session.Accept()

		if err != nil {
			code, _ := muxado.GetError(err)

			if code == muxado.SessionClosed || code == muxado.PeerEOF {
				return
			}
		}

		onetCon, err := onet.ToOnetConnWithAddr(conn, network, underlying.LocalAddr(), underlying.RemoteAddr())

		acceptChain <- &muxConn{
			conn: onetCon,
			err:  err,
		}
	}
}

func (transport *muxTransport) Server(network *onet.OverlayNetwork, conn onet.Conn, chainOffset int) (onet.Conn, error) {

	transport.RLock()
	defer transport.RUnlock()
	acceptChan, ok := transport.accpetChan[network.Addr.String()]

	serverSession := muxado.Server(conn, nil)

	sessionConn, err := serverSession.Accept()

	if err != nil {
		return nil, errors.Wrap(onet.ErrMuxNotFound, "mux %s session %s accept error", network.MuxAddrs[chainOffset], network.Addr)
	}

	if ok {
		go transport.doAccept(network, conn, serverSession, acceptChan)
	}

	return onet.ToOnetConnWithAddr(sessionConn, network, conn.LocalAddr(), conn.RemoteAddr())
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
