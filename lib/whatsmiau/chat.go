package whatsmiau

import (
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/proto"
)

type ReadMessageRequest struct {
	MessageIDs []string   `json:"message_ids"`
	InstanceID string     `json:"instance_id"`
	RemoteJID  *types.JID `json:"remote_jid"`
	Sender     *types.JID `json:"sender"`
}

func (s *Whatsmiau) ReadMessage(data *ReadMessageRequest) error {
	client, ok := s.clients.Load(data.InstanceID)
	if !ok {
		return whatsmeow.ErrClientIsNil
	}

	sender := *data.RemoteJID
	if data.Sender != nil {
		sender = *data.Sender
	}

	return client.MarkRead(context.TODO(), data.MessageIDs, time.Now(), *data.RemoteJID, sender)
}

type ChatPresenceRequest struct {
	InstanceID string                  `json:"instance_id"`
	RemoteJID  *types.JID              `json:"remote_jid"`
	Presence   types.ChatPresence      `json:"presence"`
	Media      types.ChatPresenceMedia `json:"media"`
}

func (s *Whatsmiau) ChatPresence(data *ChatPresenceRequest) error {
	client, ok := s.clients.Load(data.InstanceID)
	if !ok {
		return whatsmeow.ErrClientIsNil
	}

	return client.SendChatPresence(context.TODO(), *data.RemoteJID, data.Presence, data.Media)
}

type NumberExistsRequest struct {
	InstanceID string   `json:"instance_id"`
	Numbers    []string `json:"numbers"`
}

type NumberExistsResponse []Exists

type Exists struct {
	Exists bool   `json:"exists"`
	Jid    string `json:"jid"`
	Lid    string `json:"lid"`
	Number string `json:"number"`
}

func (s *Whatsmiau) NumberExists(ctx context.Context, data *NumberExistsRequest) (NumberExistsResponse, error) {
	client, ok := s.clients.Load(data.InstanceID)
	if !ok {
		return nil, whatsmeow.ErrClientIsNil
	}

	resp, err := client.IsOnWhatsApp(context.TODO(), data.Numbers)
	if err != nil {
		return nil, err
	}

	var results []Exists
	for _, item := range resp {
		jid, lid := s.GetJidLid(ctx, data.InstanceID, item.JID)

		results = append(results, Exists{
			Exists: item.IsIn,
			Jid:    jid,
			Lid:    lid,
			Number: item.Query,
		})
	}

	return results, nil
}

func (s *Whatsmiau) resolveJID(ctx context.Context, client *whatsmeow.Client, jid types.JID) types.JID {
	if jid.Server != types.DefaultUserServer {
		return jid
	}

	alternate := buildBrazilianAlternate(jid.User)
	if alternate == "" {
		return jid
	}

	resp, err := client.IsOnWhatsApp(ctx, []string{jid.User, alternate})
	if err != nil {
		zap.L().Warn("resolveJID: failed to check number on WhatsApp", zap.String("number", jid.User), zap.Error(err))
		return jid
	}

	for _, item := range resp {
		if item.IsIn {
			resolved := jid
			resolved.User = item.JID.User
			if resolved.User != jid.User {
				zap.L().Debug("resolveJID: brazilian number resolved", zap.String("from", jid.User), zap.String("to", resolved.User))
			}
			return resolved
		}
	}

	return jid
}

type DeleteMessageForEveryoneRequest struct {
	InstanceID     string     `json:"instance_id"`
	RemoteJID      *types.JID `json:"remote_jid"`
	MessageID      string     `json:"message_id"`
	FromMe         bool       `json:"from_me"`
	ParticipantJID *types.JID `json:"participant_jid,omitempty"`
}

func (s *Whatsmiau) DeleteMessageForEveryone(ctx context.Context, req *DeleteMessageForEveryoneRequest) error {
	client, ok := s.clients.Load(req.InstanceID)
	if !ok {
		return whatsmeow.ErrClientIsNil
	}
	if client.Store == nil || client.Store.ID == nil {
		return fmt.Errorf("device is not connected")
	}

	chat := s.resolveJID(ctx, client, *req.RemoteJID)

	var sender types.JID
	if req.FromMe {
		if chat.Server == types.GroupServer {
			sender = client.Store.ID.ToNonAD()
		} else {
			sender = types.EmptyJID
		}
	} else if chat.Server == types.GroupServer {
		sender = s.resolveJID(ctx, client, *req.ParticipantJID)
	} else {
		sender = chat
	}

	msg := client.BuildRevoke(chat, sender, types.MessageID(req.MessageID))
	_, err := client.SendMessage(ctx, chat, msg)
	return err
}

type EditMessageRequest struct {
	InstanceID string     `json:"instance_id"`
	RemoteJID  *types.JID `json:"remote_jid"`
	MessageID  string     `json:"message_id"`
	NewMessage string     `json:"new_message"`
}

func (s *Whatsmiau) EditMessage(ctx context.Context, req *EditMessageRequest) error {
	client, ok := s.clients.Load(req.InstanceID)
	if !ok {
		return whatsmeow.ErrClientIsNil
	}
	if client.Store == nil || client.Store.ID == nil {
		return fmt.Errorf("device is not connected")
	}

	chat := s.resolveJID(ctx, client, *req.RemoteJID)

	newContent := &waE2E.Message{
		Conversation: proto.String(req.NewMessage),
	}

	msg := client.BuildEdit(chat, types.MessageID(req.MessageID), newContent)
	_, err := client.SendMessage(ctx, chat, msg)
	return err
}

type ReplyMessageRequest struct {
	InstanceID          string     `json:"instance_id"`
	RemoteJID           *types.JID `json:"remote_jid"`
	MessageID           string     `json:"message_id"`
	Text                string     `json:"text"`
	ParticipantJID      *types.JID `json:"participant_jid,omitempty"`
	OriginalSenderJID   *types.JID `json:"original_sender_jid,omitempty"`
}

func (s *Whatsmiau) ReplyMessage(ctx context.Context, req *ReplyMessageRequest) (*SendTextResponse, error) {
	client, ok := s.clients.Load(req.InstanceID)
	if !ok {
		return nil, whatsmeow.ErrClientIsNil
	}
	if client.Store == nil || client.Store.ID == nil {
		return nil, fmt.Errorf("device is not connected")
	}

	chat := s.resolveJID(ctx, client, *req.RemoteJID)

	// Participant in ContextInfo = sender of the original message
	// For groups: the participant who sent the original message
	// For 1:1: the other person's JID (or empty if replying to own message)
	var participant string
	if req.OriginalSenderJID != nil {
		participant = req.OriginalSenderJID.String()
	} else if chat.Server == types.GroupServer && req.ParticipantJID != nil {
		participant = req.ParticipantJID.String()
	}

	contextInfo := &waE2E.ContextInfo{
		StanzaID: proto.String(req.MessageID),
	}
	if participant != "" {
		contextInfo.Participant = proto.String(participant)
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(req.Text),
			ContextInfo: contextInfo,
		},
	}

	res, err := client.SendMessage(ctx, chat, msg)
	if err != nil {
		return nil, err
	}

	return &SendTextResponse{
		ID:        res.ID,
		CreatedAt: res.Timestamp,
	}, nil
}
