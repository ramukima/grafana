package notifiers

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/services/notifications"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "webex",
		Name:        "Cisco Webex",
		Description: "Sends notifications using Incoming Webhook connector to Cisco Webex",
		Heading:     "Webex settings",
		Factory:     NewWebexNotifier,
		Options: []alerting.NotifierOption{
			{
				Label:        "Cisco Webex Incoming Webhook URL",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "https://webexapis.com/v1/webhooks/incoming/<room-id>",
				PropertyName: "webhook_url",
				Required:     true,
			},
			{
				Label:        "Message Content",
				Description:  "Message content template",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				PropertyName: "content",
			},
		},
	})
}

// NewWebexNotifier is the constructor for Webex notifier.
func NewWebexNotifier(model *models.AlertNotification, _ alerting.GetDecryptedValueFn, ns notifications.Service) (alerting.Notifier, error) {
	webhookURL := model.Settings.Get("webhook_url").MustString()
	if webhookURL == "" {
		return nil, alerting.ValidationError{Reason: "Could not find webhook_url property in settings"}
	}

	content := model.Settings.Get("content").MustString()

	return &WebexNotifier{
		NotifierBase: NewNotifierBase(model, ns),
		WebhookURL:   webhookURL,
		Content:      content,
		log:          log.New("alerting.notifier.webex"),
	}, nil
}

// WebexNotifier is responsible for sending
// alert notifications to Cisco Webex.
type WebexNotifier struct {
	NotifierBase
	WebhookURL string
	Content    string
	log        log.Logger
}

// Notify send an alert notification to Cisco Webex.
func (wn *WebexNotifier) Notify(evalContext *alerting.EvalContext) error {
	wn.log.Info("Executing webex notification", "ruleId", evalContext.Rule.ID, "notification", wn.Name)

	// Determine emoji
	stateEmoji := ""
	switch evalContext.Rule.State {
	case models.AlertStateOK:
		stateEmoji = "\u2705 " // Check Mark Button
	case models.AlertStateNoData:
		stateEmoji = "\u2753\uFE0F " // Question Mark
	case models.AlertStateAlerting:
		stateEmoji = "\u26A0\uFE0F " // Warning sign
	default:
		// Handle other cases?
	}

	// Build message
	message := fmt.Sprintf("%s%s\n\n*State:* %s\n*Message:* %s\n",
		stateEmoji, evalContext.GetNotificationTitle(),
		evalContext.Rule.Name, evalContext.Rule.Message)

	body := simplejson.New()

	if wn.Content != "" {
		body.Set("markdown", message)
	}

	data, _ := json.Marshal(&body)
	cmd := &models.SendWebhookSync{
		Url:         wn.WebhookURL,
		Body:        string(data),
		ContentType: "application/json; charset=utf-8",
		HttpMethod:  "POST",
		HttpHeader:  map[string]string{},
	}

	if err := wn.NotificationService.SendWebhookSync(evalContext.Ctx, cmd); err != nil {
		wn.log.Error("Failed to send webex notification", "error", err, "webhook", wn.Name)
		return err
	}

	return nil
}
