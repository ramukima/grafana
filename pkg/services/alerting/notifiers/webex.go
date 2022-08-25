package notifiers

import (
	"encoding/json"
	"fmt"

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
				Label:        "URL",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "Webex incoming webhook url",
				PropertyName: "url",
				Required:     true,
			},
			{
				Label:        "WebexToken",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "Webex API Token",
				PropertyName: "webexToken",
				Secure:       true,
				Required:     true,
			},
			{
				Label:        "RoomId",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "Webex Room Id",
				PropertyName: "roomId",
				Required:     true,
			},
		},
	})
}

// NewWebexNotifier is the constructor for Webex notifier.
func NewWebexNotifier(model *models.AlertNotification, _ alerting.GetDecryptedValueFn, ns notifications.Service) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "Could not find url property in settings"}
	}

	webexApiToken := model.Settings.Get("webexToken").MustString()
	if webexApiToken == "" {
		return nil, alerting.ValidationError{Reason: "Could not find webexToken property in settings"}
	}

	webexRoomId := model.Settings.Get("roomId").MustString()
	if webexRoomId == "" {
		return nil, alerting.ValidationError{Reason: "Could not find roomId property in settings"}
	}

	return &WebexNotifier{
		NotifierBase: NewNotifierBase(model, ns),
		URL:          url,
		ApiToken:     webexApiToken,
		RoomId:       webexRoomId,
		log:          log.New("alerting.notifier.webex"),
	}, nil
}

// WebexNotifier is responsible for sending
// alert notifications to Cisco Webex.
type WebexNotifier struct {
	NotifierBase
	URL      string
	ApiToken string
	RoomId   string
	log      log.Logger
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

	body := map[string]interface{}{
		"roomId":   wn.RoomId,
		"markdown": message,
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", wn.ApiToken),
	}

	data, _ := json.Marshal(&body)
	cmd := &models.SendWebhookSync{
		Url:         wn.URL,
		Body:        string(data),
		ContentType: "application/json; charset=utf-8",
		HttpMethod:  "POST",
		HttpHeader:  headers,
	}

	if err := wn.NotificationService.SendWebhookSync(evalContext.Ctx, cmd); err != nil {
		wn.log.Error("Failed to send webex notification", "error", err, "webhook", wn.Name)
		return err
	}

	return nil
}
