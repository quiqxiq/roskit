package service

import (
	"context"
	"fmt"
	"time"
)

type PingResult struct {
	Seq        string    `json:"seq"`
	Host       string    `json:"host"`
	Size       string    `json:"size"`
	Time       string    `json:"time"`
	Status     string    `json:"status"`
	Sent       string    `json:"sent"`
	Received   string    `json:"received"`
	PacketLoss string    `json:"packet-loss,omitempty"`
	At         time.Time `json:"at"`
}

// PingStream runs /ping on the router and streams each result via the returned channel.
// If count > 0 the channel closes after count results. If count == 0 it streams until ctx is cancelled.
func (b *Bridge) PingStream(ctx context.Context, routerID, address string, count int) (<-chan PingResult, error) {
	sentence := []string{"/ping", fmt.Sprintf("=address=%s", address)}
	if count > 0 {
		sentence = append(sentence, fmt.Sprintf("=count=%d", count))
	}

	reply, err := b.dispatcher.RunListen(ctx, routerID, sentence)
	if err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	out := make(chan PingResult, 16)
	go func() {
		defer close(out)
		defer func() {
			cancelCtx, cancelFn := context.WithTimeout(context.Background(), 3*time.Second)
			reply.CancelContext(cancelCtx)
			cancelFn()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case sentence, ok := <-reply.Chan():
				if !ok {
					return
				}
				m := sentence.Map
				result := PingResult{
					Seq:        m["seq"],
					Host:       m["host"],
					Size:       m["size"],
					Time:       m["time"],
					Status:     m["status"],
					Sent:       m["sent"],
					Received:   m["received"],
					PacketLoss: m["packet-loss"],
					At:         time.Now(),
				}
				select {
				case out <- result:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}
