package notification

import (
	"context"
	"fmt"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// SMSChannel implements notification for SMS
type SMSChannel struct {
	logger *logrus.Logger
}

func NewSMSChannel(logger *logrus.Logger) *SMSChannel {
	return &SMSChannel{
		logger: logger,
	}
}

func (s *SMSChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeSMS
}

func (s *SMSChannel) Send(ctx context.Context, message *NotificationMessage) error {
	// SMS implementation would depend on the SMS provider (Twilio, Aliyun SMS, etc.)
	// For now, this is a placeholder implementation
	
	phoneNumbers, err := s.getPhoneNumbers(message.ChannelConfig)
	if err != nil {
		return fmt.Errorf("failed to get phone numbers: %w", err)
	}

	if len(phoneNumbers) == 0 {
		return fmt.Errorf("no phone numbers specified for SMS notification")
	}

	// Format SMS content (SMS has character limits)
	smsContent := s.formatSMSContent(message)

	// Log the SMS that would be sent (in a real implementation, this would call the SMS API)
	for _, phoneNumber := range phoneNumbers {
		s.logger.WithFields(logrus.Fields{
			"phone_number": phoneNumber,
			"content":      smsContent,
		}).Info("SMS notification would be sent")
	}

	// TODO: Implement actual SMS sending logic based on your SMS provider
	// Examples:
	// - Twilio: Use Twilio SDK to send SMS
	// - Aliyun SMS: Use Aliyun SMS SDK
	// - Other providers: Implement according to their API

	return nil
}

func (s *SMSChannel) Test(ctx context.Context, testMessage string) error {
	message := &NotificationMessage{
		Title:   "AlertBot Test",
		Content: testMessage,
		Level:   "info",
		ChannelConfig: map[string]interface{}{
			"phone_numbers": []string{"test_number"},
		},
	}

	return s.Send(ctx, message)
}

func (s *SMSChannel) getPhoneNumbers(config map[string]interface{}) ([]string, error) {
	phoneNumbersInterface, ok := config["phone_numbers"]
	if !ok {
		return []string{}, nil
	}

	switch phoneNumbers := phoneNumbersInterface.(type) {
	case []interface{}:
		var result []string
		for _, phoneNumber := range phoneNumbers {
			if phoneNumberStr, ok := phoneNumber.(string); ok {
				result = append(result, phoneNumberStr)
			}
		}
		return result, nil
	case []string:
		return phoneNumbers, nil
	case string:
		return []string{phoneNumbers}, nil
	default:
		return nil, fmt.Errorf("invalid phone_numbers format")
	}
}

func (s *SMSChannel) formatSMSContent(message *NotificationMessage) string {
	// SMS messages should be concise due to character limits
	content := fmt.Sprintf("[%s] %s", message.Level, message.Title)
	
	// Add basic alert info if available
	if message.Alert != nil {
		alertName := "Unknown"
		if message.Alert.Labels != nil {
			if name, exists := message.Alert.Labels["alertname"]; exists {
				if nameStr, ok := name.(string); ok {
					alertName = nameStr
				}
			}
		}
		
		content = fmt.Sprintf("[%s] %s - %s (%s)", 
			message.Alert.Severity, 
			alertName, 
			message.Alert.Status,
			message.Alert.StartsAt.Format("15:04"))
	}

	// Limit SMS content to reasonable length (most SMS providers have 160 character limit for single SMS)
	if len(content) > 150 {
		content = content[:147] + "..."
	}

	return content
}

// Example SMS provider implementations (commented out, implement as needed)

/*
// TwilioSMSProvider implements SMS sending via Twilio
type TwilioSMSProvider struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *twilio.RestClient
}

func NewTwilioSMSProvider(accountSID, authToken, fromNumber string) *TwilioSMSProvider {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	return &TwilioSMSProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		client:     client,
	}
}

func (t *TwilioSMSProvider) SendSMS(to, message string) error {
	params := &api.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(t.fromNumber)
	params.SetBody(message)

	_, err := t.client.Api.CreateMessage(params)
	return err
}
*/

/*
// AliyunSMSProvider implements SMS sending via Aliyun SMS
type AliyunSMSProvider struct {
	accessKeyID     string
	accessKeySecret string
	signName        string
	templateCode    string
	client          *dysmsapi20170525.Client
}

func NewAliyunSMSProvider(accessKeyID, accessKeySecret, signName, templateCode string) (*AliyunSMSProvider, error) {
	config := &openapi.Config{
		AccessKeyId:     &accessKeyID,
		AccessKeySecret: &accessKeySecret,
		Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
	}

	client, err := dysmsapi20170525.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &AliyunSMSProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		signName:        signName,
		templateCode:    templateCode,
		client:          client,
	}, nil
}

func (a *AliyunSMSProvider) SendSMS(phoneNumber, message string) error {
	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  &phoneNumber,
		SignName:      &a.signName,
		TemplateCode:  &a.templateCode,
		TemplateParam: &message,
	}

	_, err := a.client.SendSms(sendSmsRequest)
	return err
}
*/