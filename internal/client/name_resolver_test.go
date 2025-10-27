package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestResolveChatNameUsesContactFullName(t *testing.T) {
	t.Parallel()

	w := &WAClient{
		contactLookup: func(ctx context.Context, user types.JID) (types.ContactInfo, error) {
			assert.Equal(t, types.NewJID("1234", types.DefaultUserServer), user)
			return types.ContactInfo{
				Found:    true,
				FullName: "Alice Smith",
			}, nil
		},
	}

	name := w.ResolveChatName(context.Background(), "1234@s.whatsapp.net", nil)
	assert.Equal(t, "Alice Smith", name)
}

func TestResolveChatNameUsesGroupInfo(t *testing.T) {
	t.Parallel()

	w := &WAClient{
		groupInfoLookup: func(ctx context.Context, jid types.JID) (*types.GroupInfo, error) {
			assert.Equal(t, types.NewJID("120363296603494645", types.GroupServer), jid)
			return &types.GroupInfo{
				GroupName: types.GroupName{Name: "Trail Runners"},
			}, nil
		},
	}

	name := w.ResolveChatName(context.Background(), "120363296603494645@g.us", nil)
	assert.Equal(t, "Trail Runners", name)
}

func TestResolveChatNameFallsBackToPushName(t *testing.T) {
	t.Parallel()

	w := &WAClient{}
	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: types.NewJID("5678", types.DefaultUserServer),
			},
			PushName: "Bobby Tables",
		},
	}

	name := w.ResolveChatName(context.Background(), "5678@s.whatsapp.net", msg)
	assert.Equal(t, "Bobby Tables", name)
}

func TestResolveChatNameFallsBackToJID(t *testing.T) {
	t.Parallel()

	w := &WAClient{}
	name := w.ResolveChatName(context.Background(), "status@broadcast", nil)
	assert.Equal(t, "status@broadcast", name)
}
