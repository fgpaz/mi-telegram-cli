package tg

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPeerNotFound            = errors.New("peer not found")
	ErrPeerAmbiguous           = errors.New("peer ambiguous")
	ErrWaitTimeout             = errors.New("wait timeout")
	ErrUnauthorized            = errors.New("unauthorized profile")
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	ErrTwoFactorRequired       = errors.New("two factor password required")
	ErrInvalidPhoneNumber      = errors.New("invalid phone number")
	ErrAuthQRTimeout           = errors.New("auth qr timeout")
	ErrMessageNotFound         = errors.New("message not found")
	ErrButtonNotFound          = errors.New("button not found")
	ErrButtonAmbiguous         = errors.New("button ambiguous")
	ErrButtonUnsupported       = errors.New("button unsupported")
	ErrButtonPasswordRequired  = errors.New("button password required")
)

type LoginMethod string

const (
	LoginMethodCode LoginMethod = "code"
	LoginMethodQR   LoginMethod = "qr"
)

type RuntimeConfig struct {
	APIID   int
	APIHash string
}

type SessionRef struct {
	ProfileID   string
	SessionPath string
	StorageRoot string
}

type LoginRequest struct {
	Method                   LoginMethod
	ProfileID                string
	PhoneNumber              string
	VerificationCode         string
	TwoFactorPassword        string
	Timeout                  time.Duration
	OnQRCode                 func(QRLoginToken) error
	RequestVerificationCode  func() (string, error)
	RequestTwoFactorPassword func() (string, error)
}

type LoginResult struct {
	AccountSummary AccountSummary
	Session        []byte
}

type LoginInputError struct {
	Field   string
	Message string
	Err     error
}

func (e *LoginInputError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "invalid login input"
}

func (e *LoginInputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type QRLoginToken struct {
	URL       string
	ExpiresAt time.Time
}

type AccountSummary struct {
	ID          any    `json:"id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName"`
	PhoneMasked string `json:"phoneMasked,omitempty"`
	IsBot       bool   `json:"isBot"`
}

type DialogSummary struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username,omitempty"`
}

type Peer struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username,omitempty"`
	Resolved    any    `json:"-"`
}

type MessageSummary struct {
	ID          int64                 `json:"id"`
	Direction   string                `json:"direction"`
	Text        string                `json:"text"`
	SentAtUTC   time.Time             `json:"sentAtUtc"`
	Attachments []AttachmentSummary   `json:"attachments"`
	Buttons     []InlineButtonSummary `json:"buttons"`
}

type SendResult struct {
	MessageID int64     `json:"messageId"`
	SentAtUTC time.Time `json:"sentAtUtc"`
}

type AttachmentSummary struct {
	Kind    string         `json:"kind"`
	Summary string         `json:"summary"`
	Details map[string]any `json:"details,omitempty"`
}

type InlineButtonSummary struct {
	Index            int    `json:"index"`
	Row              int    `json:"row"`
	Col              int    `json:"col"`
	Kind             string `json:"kind"`
	Text             string `json:"text"`
	Actionable       bool   `json:"actionable"`
	URL              string `json:"url,omitempty"`
	RequiresPassword bool   `json:"requiresPassword,omitempty"`
}

type CallbackAnswerSummary struct {
	Message   string `json:"message,omitempty"`
	Alert     bool   `json:"alert"`
	HasURL    bool   `json:"hasUrl"`
	URL       string `json:"url,omitempty"`
	NativeUI  bool   `json:"nativeUi"`
	CacheTime int    `json:"cacheTime"`
}

type PressButtonResult struct {
	Action         string                 `json:"action"`
	Button         InlineButtonSummary    `json:"button"`
	URL            string                 `json:"url,omitempty"`
	CallbackAnswer *CallbackAnswerSummary `json:"callbackAnswer,omitempty"`
}

type ListDialogsRequest struct {
	Query string
	Limit int
}

type ResolvePeerRequest struct {
	Query string
}

type ReadMessagesRequest struct {
	Peer    Peer
	Limit   int
	AfterID int64
}

type SendMessageRequest struct {
	Peer Peer
	Text string
}

type WaitMessageRequest struct {
	Peer    Peer
	AfterID int64
	Timeout time.Duration
}

type PressButtonRequest struct {
	Peer           Peer
	MessageID      int64
	ButtonIndex    int
	HasButtonIndex bool
	ButtonText     string
}

type MarkReadRequest struct {
	Peer Peer
}

type Client interface {
	Login(context.Context, RuntimeConfig, LoginRequest) (LoginResult, error)
	GetMe(context.Context, RuntimeConfig, SessionRef) (AccountSummary, error)
	ListDialogs(context.Context, RuntimeConfig, SessionRef, ListDialogsRequest) ([]DialogSummary, error)
	ResolvePeer(context.Context, RuntimeConfig, SessionRef, ResolvePeerRequest) (Peer, error)
	ReadMessages(context.Context, RuntimeConfig, SessionRef, ReadMessagesRequest) ([]MessageSummary, error)
	SendMessage(context.Context, RuntimeConfig, SessionRef, SendMessageRequest) (SendResult, error)
	WaitMessage(context.Context, RuntimeConfig, SessionRef, WaitMessageRequest) (MessageSummary, error)
	PressButton(context.Context, RuntimeConfig, SessionRef, PressButtonRequest) (PressButtonResult, error)
	MarkRead(context.Context, RuntimeConfig, SessionRef, MarkReadRequest) error
}
