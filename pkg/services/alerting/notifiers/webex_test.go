package notifiers

import (
	"testing"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/models"
	encryptionservice "github.com/grafana/grafana/pkg/services/encryption/service"

	"github.com/stretchr/testify/require"
)

func TestWebexNotifier(t *testing.T) {
	encryptionService := encryptionservice.SetupTestService(t)

	t.Run("Parsing alert notification from settings", func(t *testing.T) {
		t.Run("empty settings should return error", func(t *testing.T) {
			json := `{ }`

			settingsJSON, _ := simplejson.NewJson([]byte(json))
			model := &models.AlertNotification{
				Name:     "ops",
				Type:     "webex",
				Settings: settingsJSON,
			}

			_, err := NewWebexNotifier(model, encryptionService.GetDecryptedValue, nil)
			require.Error(t, err)
		})

		t.Run("from settings", func(t *testing.T) {
			json := `
				{
          "webhook_url": "https://webexapis.com/v1/webhooks/incoming/room-id"
				}`

			settingsJSON, _ := simplejson.NewJson([]byte(json))
			model := &models.AlertNotification{
				Name:     "ops",
				Type:     "webex",
				Settings: settingsJSON,
			}

			not, err := NewWebexNotifier(model, encryptionService.GetDecryptedValue, nil)
			webexNotifier := not.(*WebexNotifier)

			require.Nil(t, err)
			require.Equal(t, "ops", webexNotifier.Name)
			require.Equal(t, "webex", webexNotifier.Type)
			require.Equal(t, "https://webexapis.com/v1/webhooks/incoming/room-id", webexNotifier.WebhookURL)
		})
	})
}
