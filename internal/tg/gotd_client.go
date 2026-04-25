package tg

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	gotdsession "github.com/gotd/td/session"
	gotdtelegram "github.com/gotd/td/telegram"
	gotdauth "github.com/gotd/td/telegram/auth"
	gotdqrlogin "github.com/gotd/td/telegram/auth/qrlogin"
	gotdmessage "github.com/gotd/td/telegram/message"
	gotdunpack "github.com/gotd/td/telegram/message/unpack"
	gotdquery "github.com/gotd/td/telegram/query"
	gotddialogs "github.com/gotd/td/telegram/query/dialogs"
	gotduploader "github.com/gotd/td/telegram/uploader"
	gtraw "github.com/gotd/td/tg"
)

const (
	dialogBatchSize = 100
	waitPollPeriod  = time.Second
	waitReadLimit   = 100
)

var errStopCollect = errors.New("stop collect")

type GotdClient struct{}

type dialogCandidate struct {
	summary   DialogSummary
	inputPeer gtraw.InputPeerClass
}

type inlineButtonOption struct {
	Summary          InlineButtonSummary
	CallbackData     []byte
	SupportsCallback bool
	Game             bool
}

func NewGotdClient() *GotdClient {
	return &GotdClient{}
}

func (c *GotdClient) Login(ctx context.Context, runtime RuntimeConfig, req LoginRequest) (LoginResult, error) {
	switch req.Method {
	case "", LoginMethodCode:
		return c.loginByCode(ctx, runtime, req)
	case LoginMethodQR:
		return c.loginByQR(ctx, runtime, req)
	default:
		return LoginResult{}, fmt.Errorf("unsupported login method %q", req.Method)
	}
}

func (c *GotdClient) loginByCode(ctx context.Context, runtime RuntimeConfig, req LoginRequest) (LoginResult, error) {
	tempFile, err := os.CreateTemp("", "mi-telegram-cli-session-*.bin")
	if err != nil {
		return LoginResult{}, err
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()
	_ = os.Remove(tempPath)
	defer func() { _ = os.Remove(tempPath) }()

	client := newClient(runtime, tempPath)

	var self *gtraw.User
	if err := client.Run(ctx, func(runCtx context.Context) error {
		authClient := client.Auth()
		sentCode, err := authClient.SendCode(runCtx, strings.TrimSpace(req.PhoneNumber), gotdauth.SendCodeOptions{})
		if err != nil {
			return mapLoginError(err)
		}

		verificationCode, err := resolveVerificationCode(req)
		if err != nil {
			return err
		}

		switch code := sentCode.(type) {
		case *gtraw.AuthSentCode:
			if err := completeLogin(runCtx, authClient, req, verificationCode, code.PhoneCodeHash); err != nil {
				return err
			}
		case *gtraw.AuthSentCodeSuccess:
			// Fresh temp session should not normally hit this path, but it is safe.
		default:
			return fmt.Errorf("unexpected auth sent code type %T", sentCode)
		}

		self, err = client.Self(runCtx)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return LoginResult{}, err
	}

	sessionBytes, err := os.ReadFile(tempPath)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccountSummary: mapUserToAccountSummary(self),
		Session:        sessionBytes,
	}, nil
}

func resolveVerificationCode(req LoginRequest) (string, error) {
	verificationCode := strings.TrimSpace(req.VerificationCode)
	if verificationCode != "" {
		return verificationCode, nil
	}
	if req.RequestVerificationCode == nil {
		return "", &LoginInputError{
			Field:   "code",
			Message: "verification code is required",
		}
	}

	verificationCode, err := req.RequestVerificationCode()
	if err != nil {
		return "", &LoginInputError{
			Field:   "code",
			Message: fmt.Sprintf("failed to read verification code: %v", err),
			Err:     err,
		}
	}
	verificationCode = strings.TrimSpace(verificationCode)
	if verificationCode == "" {
		return "", &LoginInputError{
			Field:   "code",
			Message: "verification code is required",
		}
	}
	return verificationCode, nil
}

func resolveTwoFactorPassword(req LoginRequest) (string, error) {
	password := strings.TrimSpace(req.TwoFactorPassword)
	if password != "" {
		return password, nil
	}
	if req.RequestTwoFactorPassword == nil {
		return "", ErrTwoFactorRequired
	}

	password, err := req.RequestTwoFactorPassword()
	if err != nil {
		return "", &LoginInputError{
			Field:   "password",
			Message: fmt.Sprintf("failed to read password: %v", err),
			Err:     err,
		}
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return "", ErrTwoFactorRequired
	}
	return password, nil
}

func (c *GotdClient) loginByQR(ctx context.Context, runtime RuntimeConfig, req LoginRequest) (LoginResult, error) {
	tempFile, err := os.CreateTemp("", "mi-telegram-cli-session-*.bin")
	if err != nil {
		return LoginResult{}, err
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()
	_ = os.Remove(tempPath)
	defer func() { _ = os.Remove(tempPath) }()

	dispatcher := gtraw.NewUpdateDispatcher()
	loggedIn := gotdqrlogin.OnLoginToken(dispatcher)
	client := newQRLoginClient(runtime, tempPath, dispatcher)

	var self *gtraw.User
	loginCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	if err := client.Run(loginCtx, func(runCtx context.Context) error {
		authorization, err := client.QR().Auth(runCtx, loggedIn, func(_ context.Context, token gotdqrlogin.Token) error {
			if req.OnQRCode == nil {
				return nil
			}
			return req.OnQRCode(QRLoginToken{
				URL:       token.URL(),
				ExpiresAt: token.Expires(),
			})
		})
		if err != nil {
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				return ErrAuthQRTimeout
			}
			return err
		}

		user, ok := authorization.User.AsNotEmpty()
		if !ok {
			return fmt.Errorf("unexpected auth user type %T", authorization.User)
		}
		self = user
		return nil
	}); err != nil {
		if errors.Is(loginCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return LoginResult{}, ErrAuthQRTimeout
		}
		return LoginResult{}, err
	}

	sessionBytes, err := os.ReadFile(tempPath)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccountSummary: mapUserToAccountSummary(self),
		Session:        sessionBytes,
	}, nil
}

func (c *GotdClient) GetMe(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef) (AccountSummary, error) {
	var summary AccountSummary
	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(_ context.Context, _ *gotdtelegram.Client, _ *gtraw.Client, self *gtraw.User) error {
		summary = mapUserToAccountSummary(self)
		return nil
	})
	return summary, err
}

func (c *GotdClient) ListDialogs(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req ListDialogsRequest) ([]DialogSummary, error) {
	candidates, err := c.listDialogs(ctx, runtime, sessionRef, req.Query, req.Limit)
	if err != nil {
		return nil, err
	}

	items := make([]DialogSummary, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, candidate.summary)
	}

	return items, nil
}

func (c *GotdClient) ResolvePeer(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req ResolvePeerRequest) (Peer, error) {
	candidates, err := c.listDialogs(ctx, runtime, sessionRef, "", 0)
	if err != nil {
		return Peer{}, err
	}

	return resolvePeerFromCandidates(req.Query, candidates)
}

func (c *GotdClient) ReadMessages(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req ReadMessagesRequest) ([]MessageSummary, error) {
	var items []MessageSummary
	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		inputPeer, err := inputPeerFromPeer(req.Peer)
		if err != nil {
			return err
		}

		items, err = readMessages(runCtx, api, inputPeer, req.Limit, req.AfterID)
		return err
	})
	return items, err
}

func (c *GotdClient) SendMessage(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req SendMessageRequest) (SendResult, error) {
	var result SendResult
	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		inputPeer, err := inputPeerFromPeer(req.Peer)
		if err != nil {
			return err
		}

		updates, err := gotdmessage.NewSender(api).To(inputPeer).Text(runCtx, req.Text)
		if err != nil {
			return err
		}

		id, err := gotdunpack.MessageID(updates, nil)
		if err != nil {
			return err
		}

		sentAtUTC := time.Now().UTC()
		if message, err := gotdunpack.Message(updates, nil); err == nil {
			sentAtUTC = time.Unix(int64(message.Date), 0).UTC()
		}

		result = SendResult{
			MessageID: int64(id),
			SentAtUTC: sentAtUTC,
		}
		return nil
	})
	return result, err
}

func (c *GotdClient) SendPhoto(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req SendPhotoRequest) (SendResult, error) {
	var result SendResult
	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		inputPeer, err := inputPeerFromPeer(req.Peer)
		if err != nil {
			return err
		}

		uploadedFile, err := gotduploader.NewUploader(api).FromPath(runCtx, req.FilePath)
		if err != nil {
			return fmt.Errorf("upload photo: %w", err)
		}

		randomID, err := newRandomID()
		if err != nil {
			return err
		}

		updates, err := api.MessagesSendMedia(runCtx, &gtraw.MessagesSendMediaRequest{
			Peer: inputPeer,
			Media: &gtraw.InputMediaUploadedPhoto{
				File: uploadedFile,
			},
			Message:  req.Caption,
			RandomID: randomID,
		})
		if err != nil {
			return err
		}

		id, err := gotdunpack.MessageID(updates, nil)
		if err != nil {
			return err
		}

		sentAtUTC := time.Now().UTC()
		if message, err := gotdunpack.Message(updates, nil); err == nil {
			sentAtUTC = time.Unix(int64(message.Date), 0).UTC()
		}

		result = SendResult{
			MessageID: int64(id),
			SentAtUTC: sentAtUTC,
		}
		return nil
	})
	return result, err
}

func newRandomID() (int64, error) {
	var buf [8]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return 0, fmt.Errorf("generate random id: %w", err)
	}
	return int64(binary.LittleEndian.Uint64(buf[:])), nil
}

func (c *GotdClient) WaitMessage(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req WaitMessageRequest) (MessageSummary, error) {
	var result MessageSummary
	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		inputPeer, err := inputPeerFromPeer(req.Peer)
		if err != nil {
			return err
		}

		waitCtx, cancel := context.WithTimeout(runCtx, req.Timeout)
		defer cancel()

		ticker := time.NewTicker(waitPollPeriod)
		defer ticker.Stop()

		for {
			items, err := readMessages(waitCtx, api, inputPeer, waitReadLimit, req.AfterID)
			if err != nil {
				if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
					return ErrWaitTimeout
				}
				return err
			}

			for _, item := range items {
				if item.Direction == "incoming" && item.ID > req.AfterID {
					result = item
					return nil
				}
			}

			select {
			case <-waitCtx.Done():
				if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
					return ErrWaitTimeout
				}
				return waitCtx.Err()
			case <-ticker.C:
			}
		}
	})
	return result, err
}

func (c *GotdClient) PressButton(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req PressButtonRequest) (PressButtonResult, error) {
	var result PressButtonResult
	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		inputPeer, err := inputPeerFromPeer(req.Peer)
		if err != nil {
			return err
		}

		message, err := getMessageByID(runCtx, api, inputPeer, int(req.MessageID))
		if err != nil {
			return err
		}

		_, buttons := inlineButtonsFromReplyMarkup(message.ReplyMarkup)
		selected, err := selectInlineButton(buttons, req)
		if err != nil {
			return err
		}

		switch selected.Summary.Kind {
		case "callback":
			if selected.Summary.RequiresPassword {
				return ErrButtonPasswordRequired
			}
			answer, err := api.MessagesGetBotCallbackAnswer(runCtx, &gtraw.MessagesGetBotCallbackAnswerRequest{
				Peer:  inputPeer,
				MsgID: int(req.MessageID),
				Data:  selected.CallbackData,
			})
			if err != nil {
				return err
			}
			result = PressButtonResult{
				Action:         "callback",
				Button:         selected.Summary,
				CallbackAnswer: callbackAnswerSummaryFromResponse(answer),
			}
			return nil
		case "url":
			result = PressButtonResult{
				Action: "url",
				Button: selected.Summary,
				URL:    selected.Summary.URL,
			}
			return nil
		default:
			return ErrButtonUnsupported
		}
	})
	return result, err
}

func (c *GotdClient) MarkRead(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, req MarkReadRequest) error {
	return c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		inputPeer, err := inputPeerFromPeer(req.Peer)
		if err != nil {
			return err
		}

		items, err := readMessages(runCtx, api, inputPeer, 1, 0)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		_, err = api.MessagesReadHistory(runCtx, &gtraw.MessagesReadHistoryRequest{
			Peer:  inputPeer,
			MaxID: int(items[0].ID),
		})
		return err
	})
}

func (c *GotdClient) listDialogs(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, query string, limit int) ([]dialogCandidate, error) {
	items := make([]dialogCandidate, 0)

	err := c.withAuthorizedClient(ctx, runtime, sessionRef, func(runCtx context.Context, _ *gotdtelegram.Client, api *gtraw.Client, _ *gtraw.User) error {
		iter := gotdquery.GetDialogs(api).BatchSize(dialogBatchSize)
		err := iter.ForEach(runCtx, func(_ context.Context, elem gotddialogs.Elem) error {
			if elem.Deleted() {
				return nil
			}

			candidate, ok := candidateFromDialog(elem)
			if !ok {
				return nil
			}

			if !matchesDialogQuery(candidate.summary, query) {
				return nil
			}

			items = append(items, candidate)
			if limit > 0 && len(items) >= limit {
				return errStopCollect
			}

			return nil
		})
		if errors.Is(err, errStopCollect) {
			return nil
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (c *GotdClient) withAuthorizedClient(ctx context.Context, runtime RuntimeConfig, sessionRef SessionRef, fn func(context.Context, *gotdtelegram.Client, *gtraw.Client, *gtraw.User) error) error {
	client := newClient(runtime, sessionRef.SessionPath)

	return client.Run(ctx, func(runCtx context.Context) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return err
		}
		if !status.Authorized || status.User == nil {
			return ErrUnauthorized
		}

		return fn(runCtx, client, client.API(), status.User)
	})
}

func newClient(runtime RuntimeConfig, sessionPath string) *gotdtelegram.Client {
	return newClientWithOptions(runtime, sessionPath, nil, true)
}

func newQRLoginClient(runtime RuntimeConfig, sessionPath string, updateHandler gotdtelegram.UpdateHandler) *gotdtelegram.Client {
	return newClientWithOptions(runtime, sessionPath, updateHandler, false)
}

func newClientWithOptions(runtime RuntimeConfig, sessionPath string, updateHandler gotdtelegram.UpdateHandler, noUpdates bool) *gotdtelegram.Client {
	return gotdtelegram.NewClient(runtime.APIID, runtime.APIHash, gotdtelegram.Options{
		NoUpdates:      noUpdates,
		UpdateHandler:  updateHandler,
		SessionStorage: &gotdsession.FileStorage{Path: sessionPath},
	})
}

func completeLogin(ctx context.Context, authClient *gotdauth.Client, req LoginRequest, verificationCode string, phoneCodeHash string) error {
	_, err := authClient.SignIn(ctx, req.PhoneNumber, verificationCode, phoneCodeHash)
	if errors.Is(err, gotdauth.ErrPasswordAuthNeeded) {
		password, passwordErr := resolveTwoFactorPassword(req)
		if passwordErr != nil {
			return passwordErr
		}
		if _, err := authClient.Password(ctx, password); err != nil {
			return mapLoginError(err)
		}
		return nil
	}
	if err != nil {
		return mapLoginError(err)
	}
	return nil
}

func mapLoginError(err error) error {
	switch {
	case errors.Is(err, gotdauth.ErrPasswordAuthNeeded):
		return ErrTwoFactorRequired
	case gtraw.IsPhoneCodeInvalid(err), gtraw.IsPhoneCodeExpired(err):
		return ErrInvalidVerificationCode
	case gtraw.IsPhoneNumberInvalid(err):
		return ErrInvalidPhoneNumber
	default:
		return err
	}
}

func candidateFromDialog(elem gotddialogs.Elem) (dialogCandidate, bool) {
	switch peer := elem.Peer.(type) {
	case *gtraw.InputPeerUser:
		user, ok := elem.Entities.User(peer.UserID)
		if !ok || user == nil {
			return dialogCandidate{}, false
		}
		kind := "user"
		if user.Bot {
			kind = "bot"
		}
		return dialogCandidate{
			summary: DialogSummary{
				ID:          user.ID,
				Kind:        kind,
				DisplayName: userDisplayName(user),
				Username:    user.Username,
			},
			inputPeer: peer,
		}, true
	case *gtraw.InputPeerChat:
		chat, ok := elem.Entities.Chat(peer.ChatID)
		if !ok || chat == nil {
			return dialogCandidate{}, false
		}
		return dialogCandidate{
			summary: DialogSummary{
				ID:          chat.ID,
				Kind:        "group",
				DisplayName: strings.TrimSpace(chat.Title),
			},
			inputPeer: peer,
		}, true
	case *gtraw.InputPeerChannel:
		channel, ok := elem.Entities.Channel(peer.ChannelID)
		if !ok || channel == nil {
			return dialogCandidate{}, false
		}
		kind := "group"
		if channel.Broadcast {
			kind = "channel"
		}
		return dialogCandidate{
			summary: DialogSummary{
				ID:          channel.ID,
				Kind:        kind,
				DisplayName: strings.TrimSpace(channel.Title),
				Username:    channel.Username,
			},
			inputPeer: peer,
		}, true
	default:
		return dialogCandidate{}, false
	}
}

func resolvePeerFromCandidates(query string, candidates []dialogCandidate) (Peer, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return Peer{}, ErrPeerNotFound
	}

	if numericID, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		matches := filterCandidates(candidates, func(candidate dialogCandidate) bool {
			return candidate.summary.ID == numericID
		})
		if len(matches) == 1 {
			return peerFromCandidate(matches[0]), nil
		}
		if len(matches) > 1 {
			return Peer{}, ErrPeerAmbiguous
		}
	}

	handle := normalizeHandle(trimmed)
	if handle != "" {
		matches := filterCandidates(candidates, func(candidate dialogCandidate) bool {
			return strings.EqualFold(candidate.summary.Username, handle)
		})
		if len(matches) == 1 {
			return peerFromCandidate(matches[0]), nil
		}
		if len(matches) > 1 {
			return Peer{}, ErrPeerAmbiguous
		}
	}

	matches := filterCandidates(candidates, func(candidate dialogCandidate) bool {
		return strings.EqualFold(candidate.summary.DisplayName, trimmed)
	})
	if len(matches) == 1 {
		return peerFromCandidate(matches[0]), nil
	}
	if len(matches) > 1 {
		return Peer{}, ErrPeerAmbiguous
	}

	foldedQuery := strings.ToLower(trimmed)
	matches = filterCandidates(candidates, func(candidate dialogCandidate) bool {
		username := strings.ToLower(candidate.summary.Username)
		displayName := strings.ToLower(candidate.summary.DisplayName)
		return strings.Contains(username, normalizeHandle(foldedQuery)) || strings.Contains(displayName, foldedQuery)
	})
	if len(matches) == 1 {
		return peerFromCandidate(matches[0]), nil
	}
	if len(matches) > 1 {
		return Peer{}, ErrPeerAmbiguous
	}

	return Peer{}, ErrPeerNotFound
}

func filterCandidates(candidates []dialogCandidate, match func(dialogCandidate) bool) []dialogCandidate {
	filtered := make([]dialogCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if match(candidate) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func peerFromCandidate(candidate dialogCandidate) Peer {
	return Peer{
		ID:          candidate.summary.ID,
		Kind:        candidate.summary.Kind,
		DisplayName: candidate.summary.DisplayName,
		Username:    candidate.summary.Username,
		Resolved:    candidate.inputPeer,
	}
}

func matchesDialogQuery(summary DialogSummary, query string) bool {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return true
	}

	foldedQuery := strings.ToLower(trimmed)
	handle := normalizeHandle(trimmed)
	return strings.Contains(strings.ToLower(summary.DisplayName), foldedQuery) ||
		strings.Contains(strings.ToLower(summary.Username), handle) ||
		strings.EqualFold(strconv.FormatInt(summary.ID, 10), trimmed)
}

func inputPeerFromPeer(peer Peer) (gtraw.InputPeerClass, error) {
	inputPeer, ok := peer.Resolved.(gtraw.InputPeerClass)
	if !ok || inputPeer == nil {
		return nil, fmt.Errorf("peer is not resolved")
	}
	return inputPeer, nil
}

func readMessages(ctx context.Context, api *gtraw.Client, inputPeer gtraw.InputPeerClass, limit int, afterID int64) ([]MessageSummary, error) {
	resp, err := api.MessagesGetHistory(ctx, &gtraw.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	messages := extractMessages(resp)
	messages = messages.SortStable(func(left, right gtraw.MessageClass) bool {
		return left.GetID() > right.GetID()
	})

	items := make([]MessageSummary, 0, limit)
	for _, message := range messages {
		summary, ok := messageSummaryFromClass(message)
		if !ok {
			continue
		}
		if summary.ID <= afterID {
			continue
		}

		items = append(items, summary)
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}

func getMessageByID(ctx context.Context, api *gtraw.Client, inputPeer gtraw.InputPeerClass, messageID int) (*gtraw.Message, error) {
	var (
		resp gtraw.MessagesMessagesClass
		err  error
	)

	switch peer := inputPeer.(type) {
	case *gtraw.InputPeerChannel:
		resp, err = api.ChannelsGetMessages(ctx, &gtraw.ChannelsGetMessagesRequest{
			Channel: &gtraw.InputChannel{
				ChannelID:  peer.ChannelID,
				AccessHash: peer.AccessHash,
			},
			ID: []gtraw.InputMessageClass{
				&gtraw.InputMessageID{ID: messageID},
			},
		})
	default:
		resp, err = api.MessagesGetMessages(ctx, []gtraw.InputMessageClass{
			&gtraw.InputMessageID{ID: messageID},
		})
	}
	if err != nil {
		return nil, err
	}

	for _, message := range extractMessages(resp) {
		msg, ok := message.(*gtraw.Message)
		if !ok {
			continue
		}
		if msg.ID == messageID {
			return msg, nil
		}
	}

	return nil, ErrMessageNotFound
}

func extractMessages(resp gtraw.MessagesMessagesClass) gtraw.MessageClassArray {
	switch value := resp.(type) {
	case *gtraw.MessagesMessages:
		return value.Messages
	case *gtraw.MessagesMessagesSlice:
		return value.Messages
	case *gtraw.MessagesChannelMessages:
		return value.Messages
	default:
		return nil
	}
}

func messageSummaryFromClass(message gtraw.MessageClass) (MessageSummary, bool) {
	msg, ok := message.(*gtraw.Message)
	if !ok {
		return MessageSummary{}, false
	}

	direction := "incoming"
	if msg.Out {
		direction = "outgoing"
	}

	return MessageSummary{
		ID:          int64(msg.ID),
		Direction:   direction,
		Text:        msg.Message,
		SentAtUTC:   time.Unix(int64(msg.Date), 0).UTC(),
		Attachments: attachmentSummariesFromMedia(msg.Media),
		Buttons:     buttonSummariesFromReplyMarkup(msg.ReplyMarkup),
	}, true
}

func attachmentSummariesFromMedia(media gtraw.MessageMediaClass) []AttachmentSummary {
	items := make([]AttachmentSummary, 0)
	if media == nil {
		return items
	}

	switch value := media.(type) {
	case *gtraw.MessageMediaPhoto:
		details := map[string]any{
			"spoiler": value.Spoiler,
		}
		if value.TTLSeconds > 0 {
			details["ttlSeconds"] = value.TTLSeconds
		}
		items = append(items, AttachmentSummary{
			Kind:    "photo",
			Summary: "photo attachment",
			Details: details,
		})
	case *gtraw.MessageMediaDocument:
		items = append(items, documentAttachmentSummary(value))
	case *gtraw.MessageMediaGeo, *gtraw.MessageMediaGeoLive:
		items = append(items, AttachmentSummary{Kind: "location", Summary: "location attachment"})
	case *gtraw.MessageMediaVenue:
		items = append(items, AttachmentSummary{Kind: "venue", Summary: "venue attachment"})
	case *gtraw.MessageMediaContact:
		items = append(items, AttachmentSummary{Kind: "contact", Summary: "contact attachment"})
	case *gtraw.MessageMediaPoll:
		items = append(items, AttachmentSummary{Kind: "poll", Summary: "poll attachment"})
	case *gtraw.MessageMediaWebPage:
		items = append(items, AttachmentSummary{Kind: "webpage", Summary: "webpage attachment"})
	case *gtraw.MessageMediaInvoice:
		items = append(items, AttachmentSummary{Kind: "invoice", Summary: "invoice attachment"})
	case *gtraw.MessageMediaUnsupported:
		items = append(items, AttachmentSummary{Kind: "unsupported", Summary: "unsupported attachment"})
	case *gtraw.MessageMediaDice:
		items = append(items, AttachmentSummary{Kind: "unsupported", Summary: "dice attachment"})
	case *gtraw.MessageMediaStory:
		items = append(items, AttachmentSummary{Kind: "unsupported", Summary: "story attachment"})
	case *gtraw.MessageMediaGiveaway, *gtraw.MessageMediaGiveawayResults, *gtraw.MessageMediaPaidMedia:
		items = append(items, AttachmentSummary{Kind: "unsupported", Summary: "unsupported attachment"})
	}

	return items
}

func documentAttachmentSummary(media *gtraw.MessageMediaDocument) AttachmentSummary {
	details := map[string]any{
		"video": media.Video,
		"round": media.Round,
		"voice": media.Voice,
	}
	if media.Spoiler {
		details["spoiler"] = true
	}
	if media.TTLSeconds > 0 {
		details["ttlSeconds"] = media.TTLSeconds
	}

	kind := "document"
	if media.Voice {
		kind = "voice"
	} else if media.Video || media.Round {
		kind = "video"
	}

	if document, ok := media.Document.(*gtraw.Document); ok && document != nil {
		details["mimeType"] = document.MimeType
		details["size"] = document.Size
		details["documentId"] = document.ID

		for _, attribute := range document.Attributes {
			switch value := attribute.(type) {
			case *gtraw.DocumentAttributeFilename:
				details["fileName"] = value.FileName
			case *gtraw.DocumentAttributeSticker:
				kind = "sticker"
				if value.Alt != "" {
					details["alt"] = value.Alt
				}
			case *gtraw.DocumentAttributeAudio:
				if kind == "document" {
					kind = "audio"
				}
			}
		}

		if kind == "document" && strings.HasPrefix(strings.ToLower(document.MimeType), "audio/") {
			kind = "audio"
		}
	}

	return AttachmentSummary{
		Kind:    kind,
		Summary: kind + " attachment",
		Details: details,
	}
}

func buttonSummariesFromReplyMarkup(markup gtraw.ReplyMarkupClass) []InlineButtonSummary {
	summaries, _ := inlineButtonsFromReplyMarkup(markup)
	return summaries
}

func inlineButtonsFromReplyMarkup(markup gtraw.ReplyMarkupClass) ([]InlineButtonSummary, []inlineButtonOption) {
	summaries := make([]InlineButtonSummary, 0)
	options := make([]inlineButtonOption, 0)

	replyInline, ok := markup.(*gtraw.ReplyInlineMarkup)
	if !ok || replyInline == nil {
		return summaries, options
	}

	index := 0
	for rowIndex, row := range replyInline.Rows {
		for colIndex, button := range row.Buttons {
			option, ok := inlineButtonFromClass(button, index, rowIndex, colIndex)
			if !ok {
				continue
			}
			summaries = append(summaries, option.Summary)
			options = append(options, option)
			index++
		}
	}

	return summaries, options
}

func inlineButtonFromClass(button gtraw.KeyboardButtonClass, index, row, col int) (inlineButtonOption, bool) {
	summary := InlineButtonSummary{
		Index: index,
		Row:   row,
		Col:   col,
	}

	switch value := button.(type) {
	case *gtraw.KeyboardButtonCallback:
		summary.Kind = "callback"
		summary.Text = value.Text
		summary.Actionable = true
		summary.RequiresPassword = value.RequiresPassword
		return inlineButtonOption{
			Summary:          summary,
			CallbackData:     append([]byte(nil), value.Data...),
			SupportsCallback: true,
		}, true
	case *gtraw.KeyboardButtonURL:
		summary.Kind = "url"
		summary.Text = value.Text
		summary.Actionable = true
		summary.URL = value.URL
		return inlineButtonOption{Summary: summary}, true
	case *gtraw.KeyboardButton:
		summary.Kind = "text"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonSwitchInline:
		summary.Kind = "switch-inline"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonGame:
		summary.Kind = "game"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonBuy:
		summary.Kind = "buy"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonURLAuth:
		summary.Kind = "url-auth"
		summary.Text = value.Text
		summary.URL = value.URL
	case *gtraw.KeyboardButtonWebView:
		summary.Kind = "webview"
		summary.Text = value.Text
		summary.URL = value.URL
	case *gtraw.KeyboardButtonSimpleWebView:
		summary.Kind = "simple-webview"
		summary.Text = value.Text
		summary.URL = value.URL
	case *gtraw.KeyboardButtonRequestPhone:
		summary.Kind = "request-phone"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonRequestGeoLocation:
		summary.Kind = "request-geo"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonRequestPoll:
		summary.Kind = "request-poll"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonRequestPeer:
		summary.Kind = "request-peer"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonUserProfile:
		summary.Kind = "user-profile"
		summary.Text = value.Text
	case *gtraw.KeyboardButtonCopy:
		summary.Kind = "copy"
		summary.Text = value.Text
	default:
		return inlineButtonOption{}, false
	}

	return inlineButtonOption{Summary: summary}, true
}

func selectInlineButton(buttons []inlineButtonOption, req PressButtonRequest) (inlineButtonOption, error) {
	if req.HasButtonIndex {
		for _, button := range buttons {
			if button.Summary.Index == req.ButtonIndex {
				return button, nil
			}
		}
		return inlineButtonOption{}, ErrButtonNotFound
	}

	needle := strings.TrimSpace(req.ButtonText)
	if needle == "" {
		return inlineButtonOption{}, ErrButtonNotFound
	}

	var selected *inlineButtonOption
	for idx := range buttons {
		if buttons[idx].Summary.Text != needle {
			continue
		}
		if selected != nil {
			return inlineButtonOption{}, ErrButtonAmbiguous
		}
		selected = &buttons[idx]
	}
	if selected == nil {
		return inlineButtonOption{}, ErrButtonNotFound
	}
	return *selected, nil
}

func callbackAnswerSummaryFromResponse(answer *gtraw.MessagesBotCallbackAnswer) *CallbackAnswerSummary {
	if answer == nil {
		return nil
	}
	return &CallbackAnswerSummary{
		Message:   answer.Message,
		Alert:     answer.Alert,
		HasURL:    answer.HasURL,
		URL:       answer.URL,
		NativeUI:  answer.NativeUI,
		CacheTime: answer.CacheTime,
	}
}

func mapUserToAccountSummary(user *gtraw.User) AccountSummary {
	if user == nil {
		return AccountSummary{}
	}

	return AccountSummary{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: userDisplayName(user),
		PhoneMasked: maskPhone(user.Phone),
		IsBot:       user.Bot,
	}
}

func userDisplayName(user *gtraw.User) string {
	if user == nil {
		return ""
	}

	displayName := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if displayName != "" {
		return displayName
	}
	if user.Username != "" {
		return user.Username
	}
	return strconv.FormatInt(user.ID, 10)
}

func normalizeHandle(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "@")
}

func maskPhone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	runes := []rune(value)
	digitIndexes := make([]int, 0, len(runes))
	for index, r := range runes {
		if r >= '0' && r <= '9' {
			digitIndexes = append(digitIndexes, index)
		}
	}
	if len(digitIndexes) <= 4 {
		return value
	}

	for _, index := range digitIndexes[:len(digitIndexes)-4] {
		runes[index] = '*'
	}
	return string(runes)
}
