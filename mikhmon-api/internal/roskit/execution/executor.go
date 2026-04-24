package execution

import (
	"context"
	"fmt"

	"github.com/go-routeros/routeros/v3"
)

type Executor struct {
	pool *Pool
}

func NewExecutor(pool *Pool) *Executor {
	return &Executor{pool: pool}
}

func (e *Executor) Run(ctx context.Context, routerID string, sentence ...string) (*routeros.Reply, error) {
	conn, err := e.pool.Borrow(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("executor: %w", err)
	}
	defer e.pool.Return(routerID, conn)
	return conn.RunContext(ctx, sentence...)
}

func (e *Executor) Add(ctx context.Context, routerID, path string, params map[string]string) (*routeros.Reply, error) {
	sentence := buildAdd(path, params)
	return e.Run(ctx, routerID, sentence...)
}

func (e *Executor) Set(ctx context.Context, routerID, path, id string, params map[string]string) error {
	sentence := buildSet(path, id, params)
	_, err := e.Run(ctx, routerID, sentence...)
	return err
}

func (e *Executor) Remove(ctx context.Context, routerID, path, id string) error {
	sentence := buildRemove(path, id)
	_, err := e.Run(ctx, routerID, sentence...)
	return err
}

func (e *Executor) Enable(ctx context.Context, routerID, path, id string) error {
	sentence := []string{fmt.Sprintf("/%s/enable", path), fmt.Sprintf("=numbers=%s", id)}
	_, err := e.Run(ctx, routerID, sentence...)
	return err
}

func (e *Executor) Disable(ctx context.Context, routerID, path, id string) error {
	sentence := []string{fmt.Sprintf("/%s/disable", path), fmt.Sprintf("=numbers=%s", id)}
	_, err := e.Run(ctx, routerID, sentence...)
	return err
}

func (e *Executor) ExtractID(reply *routeros.Reply) string {
	if reply.Done != nil {
		for _, pair := range reply.Done.List {
			if pair.Key == "ret" {
				return pair.Value
			}
		}
	}
	return ""
}

func buildAdd(path string, params map[string]string) []string {
	sentence := []string{fmt.Sprintf("/%s/add", path)}
	for k, v := range params {
		sentence = append(sentence, fmt.Sprintf("=%s=%s", k, v))
	}
	return sentence
}

func buildSet(path, id string, params map[string]string) []string {
	sentence := []string{fmt.Sprintf("/%s/set", path), fmt.Sprintf("=.id=%s", id)}
	for k, v := range params {
		sentence = append(sentence, fmt.Sprintf("=%s=%s", k, v))
	}
	return sentence
}

func buildRemove(path, id string) []string {
	return []string{fmt.Sprintf("/%s/remove", path), fmt.Sprintf("=.id=%s", id)}
}
