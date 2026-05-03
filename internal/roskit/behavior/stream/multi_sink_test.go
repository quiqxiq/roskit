package stream

import (
	"context"
	"errors"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureSink struct {
	events []behavior.StreamEvent
	err    error
}

func (c *captureSink) OnEvent(_ context.Context, event behavior.StreamEvent) error {
	c.events = append(c.events, event)
	return c.err
}

func makeEvent(measurement string) behavior.StreamEvent {
	return behavior.StreamEvent{
		Meta:     &command.CommandMeta{Measurement: measurement},
		RouterID: "r1",
		Fields:   map[string]string{"name": "user1"},
	}
}

func TestMultiSink_FanOut(t *testing.T) {
	s1 := &captureSink{}
	s2 := &captureSink{}
	ms := NewMultiSink([]behavior.StreamSink{s1, s2}, nil)

	event := makeEvent("hotspot_user")
	err := ms.OnEvent(context.Background(), event)

	require.NoError(t, err)
	assert.Len(t, s1.events, 1)
	assert.Len(t, s2.events, 1)
	assert.Equal(t, "hotspot_user", s1.events[0].Meta.Measurement)
}

func TestMultiSink_EmptySinks(t *testing.T) {
	ms := NewMultiSink(nil, nil)
	err := ms.OnEvent(context.Background(), makeEvent("m"))
	assert.NoError(t, err)
}

func TestMultiSink_SinkErrorDoesNotBlock(t *testing.T) {
	failing := &captureSink{err: errors.New("sink down")}
	good := &captureSink{}
	ms := NewMultiSink([]behavior.StreamSink{failing, good}, nil)

	err := ms.OnEvent(context.Background(), makeEvent("m"))

	// MultiSink never propagates individual sink errors
	assert.NoError(t, err)
	// good sink still received the event
	assert.Len(t, good.events, 1)
}

func TestMultiSink_MultipleEvents(t *testing.T) {
	s := &captureSink{}
	ms := NewMultiSink([]behavior.StreamSink{s}, nil)

	for range 5 {
		require.NoError(t, ms.OnEvent(context.Background(), makeEvent("m")))
	}
	assert.Len(t, s.events, 5)
}

func TestMultiSink_DeadEvent_Forwarded(t *testing.T) {
	s := &captureSink{}
	ms := NewMultiSink([]behavior.StreamSink{s}, nil)

	dead := behavior.StreamEvent{
		Meta:     &command.CommandMeta{Measurement: "hotspot_user"},
		RouterID: "r1",
		IsDead:   true,
		Fields:   map[string]string{".id": "*1"},
	}
	require.NoError(t, ms.OnEvent(context.Background(), dead))
	assert.True(t, s.events[0].IsDead)
}
