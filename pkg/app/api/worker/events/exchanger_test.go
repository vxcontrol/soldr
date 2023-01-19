package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExchangerSubscribe(t *testing.T) {
	type (
		subParams struct {
			serviceID uint64
			events    []EventChannelName
		}
		testCase struct {
			subscribes    subParams
			firedEvents   []subParams
			reachedNumber int
		}
	)
	testCases := []testCase{
		{
			subscribes:    subParams{1, []EventChannelName{CreateAgentsChannel}},
			firedEvents:   []subParams{{1, []EventChannelName{CreateAgentsChannel}}},
			reachedNumber: 1,
		},
		{
			subscribes:    subParams{1, []EventChannelName{CreateAgentsChannel, DeleteAgentsChannel}},
			firedEvents:   []subParams{{1, []EventChannelName{CreateAgentsChannel, DeleteAgentsChannel}}},
			reachedNumber: 2,
		},
		{
			subscribes: subParams{1, []EventChannelName{CreateAgentsChannel, DeleteAgentsChannel}},
			firedEvents: []subParams{
				{1, []EventChannelName{CreateAgentsChannel, DeleteAgentsChannel}},
				{2, []EventChannelName{CreateAgentsChannel}},
			},
			reachedNumber: 2,
		},
		{
			subscribes: subParams{1, []EventChannelName{CreateAgentsChannel, DeleteAgentsChannel}},
			firedEvents: []subParams{
				{1, []EventChannelName{CreateAgentsChannel, UpdateAgentsChannel}},
				{2, []EventChannelName{CreateAgentsChannel}},
			},
			reachedNumber: 1,
		},
		{
			subscribes: subParams{2, []EventChannelName{CreateAgentsChannel, DeleteAgentsChannel}},
			firedEvents: []subParams{
				{1, []EventChannelName{CreateAgentsChannel, CreateAgentsChannel, DeleteAgentsChannel, UpdateAgentsChannel}},
				{2, []EventChannelName{CreateAgentsChannel, UpdateAgentsChannel}},
			},
			reachedNumber: 1,
		},
	}

	for _, tc := range testCases {
		exchanger := NewExchanger()

		subscription := exchanger.Subscribe(tc.subscribes.serviceID, tc.subscribes.events...)

		go func() {
			for _, scenario := range tc.firedEvents {
				for _, e := range scenario.events {
					exchanger.fireEvents(scenario.serviceID, e, Event{Name: "some event"})
				}
			}
		}()

		var reachCounter int
		timeout := time.NewTimer(time.Second)

	AssertLoop:
		for {
			select {
			case <-timeout.C:
				require.Fail(t, "events did not reach the subscriber")
			case <-subscription.C:
				reachCounter++
				if reachCounter == tc.reachedNumber {
					break AssertLoop
				}
			}
		}
	}
}

func TestExchangerUnSubscribe(t *testing.T) {
	exchanger := NewExchanger()

	subscription1 := exchanger.Subscribe(1, CreateAgentsChannel, DeleteAgentsChannel)
	subscription2 := exchanger.Subscribe(1, CreateAgentsChannel)
	subscription3 := exchanger.Subscribe(1, CreateAgentsChannel, DeleteAgentsChannel)
	subscription4 := exchanger.Subscribe(2, CreateAgentsChannel)

	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][1], 3)
	require.Len(t, exchanger.subscriptions.data[DeleteAgentsChannel][1], 2)
	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][2], 1)

	exchanger.UnSubscribe(subscription2)

	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][1], 2)
	require.Len(t, exchanger.subscriptions.data[DeleteAgentsChannel][1], 2)
	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][2], 1)

	exchanger.UnSubscribe(subscription1)
	exchanger.UnSubscribe(subscription3)

	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][1], 0)
	require.Len(t, exchanger.subscriptions.data[DeleteAgentsChannel][1], 0)
	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][2], 1)

	exchanger.UnSubscribe(subscription4)

	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][1], 0)
	require.Len(t, exchanger.subscriptions.data[DeleteAgentsChannel][1], 0)
	require.Len(t, exchanger.subscriptions.data[CreateAgentsChannel][2], 0)
}
