package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// QREvent is sent over the channel during the QR pairing flow.
type QREvent struct {
	Type    string // "qr" | "connected" | "timeout" | "error"
	Code    string // QR string (type=="qr") or JID (type=="connected")
	Message string // error description
}

type activeEntry struct {
	client    *whatsmeow.Client
	sessionID uuid.UUID
}

// Manager owns the pool of live whatsmeow clients.
type Manager struct {
	mu      sync.RWMutex
	pool    []activeEntry          // ordered slice for round-robin
	byID    map[uuid.UUID]*whatsmeow.Client
	counter atomic.Int64
	waDB    *sqlstore.Container
	repo    *Repository
	dbURL   string
	log     *slog.Logger
}

func NewManager(dbURL string, repo *Repository, log *slog.Logger) (*Manager, error) {
	container, err := sqlstore.New(context.Background(), "pgx", dbURL, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("whatsmeow sqlstore: %w", err)
	}
	return &Manager{
		byID:  make(map[uuid.UUID]*whatsmeow.Client),
		waDB:  container,
		repo:  repo,
		dbURL: dbURL,
		log:   log,
	}, nil
}

// RestoreConnected reconnects all sessions that were previously connected.
// Call once at startup.
func (m *Manager) RestoreConnected(ctx context.Context) {
	sessions, err := m.repo.ListConnectedSessions(ctx)
	if err != nil {
		m.log.Error("wa: restore sessions", "error", err)
		return
	}
	for _, s := range sessions {
		if s.JID == "" {
			continue
		}
		jid, err := types.ParseJID(s.JID)
		if err != nil {
			continue
		}
		device, err := m.waDB.GetDevice(ctx, jid)
		if err != nil || device == nil {
			_ = m.repo.UpdateSessionStatus(ctx, s.ID, StatusDisconnected)
			continue
		}
		client := whatsmeow.NewClient(device, waLog.Noop)
		if err := client.Connect(); err != nil {
			_ = m.repo.UpdateSessionStatus(ctx, s.ID, StatusDisconnected)
			m.log.Warn("wa: reconnect failed", "jid", s.JID, "error", err)
			continue
		}
		m.addToPool(s.ID, client)
		m.log.Info("wa: restored session", "name", s.Name, "jid", s.JID)
	}
}

// StartQR initiates a QR-code pairing flow for a pending session.
// Returns immediately; events arrive on the returned channel which closes when done.
// ctx should be cancelled by the caller when the HTTP connection is closed.
func (m *Manager) StartQR(ctx context.Context, sessionID uuid.UUID) (<-chan QREvent, error) {
	device := m.waDB.NewDevice()
	client := whatsmeow.NewClient(device, waLog.Noop)

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get qr channel: %w", err)
	}

	out := make(chan QREvent, 16)

	// send is a non-blocking helper; drops the event if ctx is done (caller gone).
	send := func(evt QREvent) bool {
		select {
		case out <- evt:
			return true
		case <-ctx.Done():
			return false
		}
	}

	go func() {
		defer close(out)

		// Connect() uses client.BackgroundEventCtx (context.Background()) internally
		// so the persistent keepAlive/handler loops are not tied to our ctx.
		// The initial TCP dial can block 30–120 s if WA is unreachable — run it here
		// so the caller is never blocked and can keep sending SSE keepalive pings.
		if err := client.Connect(); err != nil {
			send(QREvent{Type: "error", Message: err.Error()})
			return
		}

		for {
			select {
			case evt, open := <-qrChan:
				if !open {
					return
				}
				switch evt.Event {
				case "code":
					if !send(QREvent{Type: "qr", Code: evt.Code}) {
						client.Disconnect()
						return
					}
				case "success":
					jid := client.Store.ID.String()
					phone := strings.Split(jid, ":")[0]
					_ = m.repo.UpdateSessionPaired(ctx, sessionID, jid, phone)
					m.addToPool(sessionID, client)
					send(QREvent{Type: "connected", Code: jid})
					return
				case "timeout":
					client.Disconnect()
					send(QREvent{Type: "timeout", Message: "QR code expired"})
					return
				default:
					if evt.Error != nil {
						client.Disconnect()
						send(QREvent{Type: "error", Message: evt.Error.Error()})
						return
					}
				}
			case <-ctx.Done():
				client.Disconnect()
				return
			}
		}
	}()

	return out, nil
}

// GetPairingCode requests a phone-number-based pairing code (alternative to QR).
func (m *Manager) GetPairingCode(ctx context.Context, sessionID uuid.UUID, phone string) (string, error) {
	device := m.waDB.NewDevice()
	client := whatsmeow.NewClient(device, waLog.Noop)

	if err := client.Connect(); err != nil {
		return "", fmt.Errorf("connect: %w", err)
	}

	code, err := client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		client.Disconnect()
		return "", fmt.Errorf("pair phone: %w", err)
	}

	client.AddEventHandler(func(evt interface{}) {
		if _, ok := evt.(*events.PairSuccess); ok {
			jid := client.Store.ID.String()
			phoneParsed := strings.Split(jid, ":")[0]
			_ = m.repo.UpdateSessionPaired(ctx, sessionID, jid, phoneParsed)
			m.addToPool(sessionID, client)
		}
	})

	return code, nil
}

// Send picks a session via round-robin and sends a text message.
// Returns the UUID of the session that was used.
func (m *Manager) Send(ctx context.Context, recipient, body string) (uuid.UUID, error) {
	entry, err := m.pick()
	if err != nil {
		return uuid.Nil, err
	}

	jid, err := types.ParseJID(normalizePhone(recipient))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid recipient: %w", err)
	}

	msg := &waProto.Message{
		Conversation: proto.String(body),
	}
	if _, err := entry.client.SendMessage(ctx, jid, msg); err != nil {
		return uuid.Nil, fmt.Errorf("send message: %w", err)
	}

	_ = m.repo.RecordUsage(ctx, entry.sessionID)
	return entry.sessionID, nil
}

// DisconnectSession disconnects and removes a session from the pool.
func (m *Manager) DisconnectSession(ctx context.Context, id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.byID[id]
	if ok {
		client.Disconnect()
		delete(m.byID, id)
		m.rebuildPool()
	}
	_ = m.repo.UpdateSessionStatus(ctx, id, StatusDisconnected)
}

// Shutdown gracefully disconnects all sessions.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.byID {
		c.Disconnect()
	}
}

// ---- internal helpers ----

func (m *Manager) addToPool(id uuid.UUID, client *whatsmeow.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[id] = client
	m.rebuildPool()

	// Watch for disconnects / bans
	client.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Disconnected:
			_ = m.repo.UpdateSessionStatus(context.Background(), id, StatusDisconnected)
			m.mu.Lock()
			delete(m.byID, id)
			m.rebuildPool()
			m.mu.Unlock()
		case *events.LoggedOut:
			_ = m.repo.UpdateSessionStatus(context.Background(), id, StatusBanned)
			m.mu.Lock()
			delete(m.byID, id)
			m.rebuildPool()
			m.mu.Unlock()
		}
	})
}

// rebuildPool re-builds the ordered slice from byID. Must be called with mu held.
func (m *Manager) rebuildPool() {
	m.pool = m.pool[:0]
	for id, c := range m.byID {
		m.pool = append(m.pool, activeEntry{client: c, sessionID: id})
	}
}

func (m *Manager) pick() (*activeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	n := len(m.pool)
	if n == 0 {
		return nil, fmt.Errorf("no connected whatsapp sessions available")
	}
	idx := int(m.counter.Add(1)-1) % n
	e := m.pool[idx]
	return &e, nil
}

// normalizePhone converts a phone number to WhatsApp JID format.
// Accepts "628123456789", "+628123456789", or "628123456789@s.whatsapp.net".
func normalizePhone(phone string) string {
	phone = strings.TrimPrefix(phone, "+")
	if strings.Contains(phone, "@") {
		return phone
	}
	return phone + "@s.whatsapp.net"
}
