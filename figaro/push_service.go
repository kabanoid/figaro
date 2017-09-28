package figaro

import (
	"context"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

// PushService responsible for push notifications
type PushService struct {
	upgrader websocket.Upgrader
	in       chan []byte
	outs     map[chan []byte]struct{}
	addCh    chan chan []byte
	removeCh chan chan []byte
}

// NewPushService creates and launches push service on the specified address
func NewPushService() *PushService {
	p := &PushService{}
	p.upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	p.in = make(chan []byte)
	p.outs = make(map[chan []byte]struct{})
	p.addCh = make(chan chan []byte)
	p.removeCh = make(chan chan []byte)
	go p.serve()
	log.Println("Push service started")
	return p
}

// In channel
func (p *PushService) In() chan<- []byte {
	return p.in
}

func (p *PushService) serve() {
	var cancel context.CancelFunc = func() {}
	for {
		select {
		case data := <-p.in:
			cancel()
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			for ch := range p.outs {
				go func(ch chan []byte) {
					select {
					case ch <- data:
					case <-ctx.Done():
						return
					}
				}(ch)
			}
		case ch := <-p.addCh:
			p.outs[ch] = struct{}{}
		case ch := <-p.removeCh:
			delete(p.outs, ch)
		}
	}
}

// Handler handels http requests. It upgrades HTTP request to WS connection and
// serves it.
func (p *PushService) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Cannot upgrade:", err)
		return
	}
	defer conn.Close()
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	in := make(chan []byte)
	p.addCh <- in
	defer func() { p.removeCh <- in }()
	for {
		msg := <-in
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Cannot write to the WS:", err)
			return
		}
	}
}
