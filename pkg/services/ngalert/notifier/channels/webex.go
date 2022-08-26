package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	ngmodels "github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/notifications"
)

// WebexNotifier is responsible for sending
// alert notifications to Webex Team Space.
type WebexNotifier struct {
	*Base
	WebhookURL string
	Content    string
	log        log.Logger
	images     ImageStore
	ns         notifications.WebhookSender
	tmpl       *template.Template
}

type WebexConfig struct {
	*NotificationChannelConfig
	WebhookURL string
	Content    string
}

func WebexFactory(fc FactoryConfig) (NotificationChannel, error) {
	cfg, err := NewWebexConfig(fc.Config, fc.DecryptFunc)
	if err != nil {
		return nil, receiverInitError{
			Reason: err.Error(),
			Cfg:    *fc.Config,
		}
	}
	return NewWebexNotifier(cfg, fc.ImageStore, fc.NotificationService, fc.Template), nil
}

func NewWebexConfig(config *NotificationChannelConfig, decryptFunc GetDecryptedValueFn) (*WebexConfig, error) {
	webhookUrl := config.Settings.Get("webhook_url").MustString()
	if webhookUrl == "" {
		return nil, errors.New("could not find Webex Webhook URL in settings")
	}
	return &WebexConfig{
		NotificationChannelConfig: config,
		WebhookURL:                webhookUrl,
		Content:                   config.Settings.Get("message").MustString(`{{ template "default.message" . }}`),
	}, nil
}

// NewWebexNotifier is the constructor for the Threema notifier
func NewWebexNotifier(config *WebexConfig, images ImageStore, ns notifications.WebhookSender, t *template.Template) *WebexNotifier {
	return &WebexNotifier{
		Base: NewBase(&models.AlertNotification{
			Uid:                   config.UID,
			Name:                  config.Name,
			Type:                  config.Type,
			DisableResolveMessage: config.DisableResolveMessage,
			Settings:              config.Settings,
		}),
		WebhookURL: config.WebhookURL,
		Content:    config.Content,
		log:        log.New("alerting.notifier.webex"),
		images:     images,
		ns:         ns,
		tmpl:       t,
	}
}

// Notify send an alert notification to Webex
func (tn *WebexNotifier) Notify(ctx context.Context, as ...*types.Alert) (bool, error) {
	tn.log.Debug("sending webex alert notification at url", tn.WebhookURL)

	var tmplErr error
	tmpl, _ := TmplText(ctx, tn.tmpl, as, tn.log, &tmplErr)

	// Determine emoji
	stateEmoji := "\u26A0\uFE0F " // Warning sign
	alerts := types.Alerts(as...)
	if alerts.Status() == model.AlertResolved {
		stateEmoji = "\u2705 " // Check Mark Button
	}

	// Build message
	message := fmt.Sprintf("%s%s\n\n*Message:*\n%s\n*URL:* %s\n",
		stateEmoji,
		tmpl(DefaultMessageTitleEmbed),
		tmpl(tn.Content),
		path.Join(tn.tmpl.ExternalURL.String(), "/alerting/list"),
	)

	if tmplErr != nil {
		tn.log.Warn("failed to template Webex message", "err", tmplErr.Error())
	}

	_ = withStoredImages(ctx, tn.log, tn.images,
		func(_ int, image ngmodels.Image) error {
			if image.URL != "" {
				message += fmt.Sprintf("*Image:* %s\n", image.URL)
			}
			return nil
		}, as...)

	body := simplejson.New()
	body.Set("markdown", message)

	data, _ := json.Marshal(&body)
	headers := map[string]string{}

	cmd := &models.SendWebhookSync{
		Url:         tn.WebhookURL,
		Body:        string(data),
		HttpMethod:  "POST",
		ContentType: "application/json; charset=utf-8",
		HttpHeader:  headers,
	}
	if err := tn.ns.SendWebhookSync(ctx, cmd); err != nil {
		tn.log.Error("Failed to send webex notification", "err", err, "webhook", tn.Name)
		return false, err
	}

	return true, nil
}

func (tn *WebexNotifier) SendResolved() bool {
	return !tn.GetDisableResolveMessage()
}
