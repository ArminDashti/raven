package push

import (
	"database/sql"
	"encoding/json"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type Sender struct {
	DB         *sql.DB
	PublicKey  string
	PrivateKey string
	Subject    string
	BaseURL    string
}

type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

func (s *Sender) Enabled() bool {
	return s != nil && s.PublicKey != "" && s.PrivateKey != ""
}

func (s *Sender) NotifyUsers(userIDs []int64, title, body, path string) {
	if !s.Enabled() || len(userIDs) == 0 {
		return
	}
	url := s.BaseURL
	if path != "" {
		url = s.BaseURL + path
	}
	payload, err := json.Marshal(Payload{Title: title, Body: body, URL: url})
	if err != nil {
		return
	}
	seen := map[int64]struct{}{}
	for _, uid := range userIDs {
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		s.sendToUser(uid, payload)
	}
}

func (s *Sender) sendToUser(userID int64, payload []byte) {
	rows, err := s.DB.Query(`
SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?`, userID)
	if err != nil {
		log.Printf("push query: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var endpoint, p256dh, auth string
		if err := rows.Scan(&id, &endpoint, &p256dh, &auth); err != nil {
			continue
		}
		sub := &webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				P256dh: p256dh,
				Auth:   auth,
			},
		}
		resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
			Subscriber:      s.Subject,
			VAPIDPublicKey:  s.PublicKey,
			VAPIDPrivateKey: s.PrivateKey,
			TTL:             60,
		})
		if err != nil {
			log.Printf("push send: %v", err)
			continue
		}
		status := resp.StatusCode
		_ = resp.Body.Close()
		if status == 404 || status == 410 {
			_, _ = s.DB.Exec(`DELETE FROM push_subscriptions WHERE id = ?`, id)
		}
	}
}
