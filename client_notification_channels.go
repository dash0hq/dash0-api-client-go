package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListNotificationChannels retrieves all notification channels.
func (c *client) ListNotificationChannels(ctx context.Context) ([]*NotificationChannelDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiNotificationChannelsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: list notification channels failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetNotificationChannel retrieves a notification channel by origin or ID.
func (c *client) GetNotificationChannel(ctx context.Context, originOrID string) (*NotificationChannelDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiNotificationChannelsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return nil, fmt.Errorf("dash0: get notification channel failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateNotificationChannel creates a new notification channel.
func (c *client) CreateNotificationChannel(ctx context.Context, channel *NotificationChannelDefinition) (*NotificationChannelDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.PostApiNotificationChannelsWithResponse(ctx, *channel)
	if err != nil {
		return nil, fmt.Errorf("dash0: create notification channel failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateNotificationChannel updates an existing notification channel.
func (c *client) UpdateNotificationChannel(ctx context.Context, originOrID string, channel *NotificationChannelDefinition) (*NotificationChannelDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.PutApiNotificationChannelsOriginOrIdWithResponse(ctx, originOrID, *channel)
	if err != nil {
		return nil, fmt.Errorf("dash0: update notification channel failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteNotificationChannel deletes a notification channel by origin or ID.
func (c *client) DeleteNotificationChannel(ctx context.Context, originOrID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiNotificationChannelsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return fmt.Errorf("dash0: delete notification channel failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListNotificationChannelsIter returns an iterator over all notification channels.
// This is a convenience wrapper around ListNotificationChannels for consistent iteration patterns.
func (c *client) ListNotificationChannelsIter(ctx context.Context) *Iter[NotificationChannelDefinition] {
	items, err := c.ListNotificationChannels(ctx)
	if err != nil {
		return newIterWithError[NotificationChannelDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripNotificationChannelServerFields removes server-generated fields from a
// notification channel definition.
func StripNotificationChannelServerFields(channel *NotificationChannelDefinition) {
	if channel == nil {
		return
	}
	if channel.Metadata.Annotations != nil {
		channel.Metadata.Annotations.Dash0ComcreatedAt = nil
		channel.Metadata.Annotations.Dash0ComupdatedAt = nil
	}
	if channel.Metadata.Labels != nil {
		channel.Metadata.Labels.Dash0Comid = nil
	}
}

// ClearNotificationChannelID removes the ID from a notification channel definition.
func ClearNotificationChannelID(channel *NotificationChannelDefinition) {
	if channel == nil {
		return
	}
	if channel.Metadata.Labels != nil {
		channel.Metadata.Labels.Dash0Comid = nil
	}
}

// GetNotificationChannelID extracts the ID from a notification channel definition.
func GetNotificationChannelID(channel *NotificationChannelDefinition) string {
	if channel == nil || channel.Metadata.Labels == nil || channel.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *channel.Metadata.Labels.Dash0Comid
}

// GetNotificationChannelName extracts the display name from a notification channel definition.
func GetNotificationChannelName(channel *NotificationChannelDefinition) string {
	if channel == nil {
		return ""
	}
	return channel.Metadata.Name
}

// GetNotificationChannelOrigin extracts the origin from a notification channel definition.
func GetNotificationChannelOrigin(channel *NotificationChannelDefinition) string {
	if channel == nil || channel.Metadata.Labels == nil || channel.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *channel.Metadata.Labels.Dash0Comorigin
}

// SetNotificationChannelID sets the dash0.com/id label on a notification channel
// definition, initializing the labels struct if needed.
func SetNotificationChannelID(channel *NotificationChannelDefinition, id string) {
	if channel == nil {
		return
	}
	if channel.Metadata.Labels == nil {
		channel.Metadata.Labels = &NotificationChannelLabels{}
	}
	channel.Metadata.Labels.Dash0Comid = &id
}

// SetNotificationChannelIDIfAbsent sets the dash0.com/id label on a notification
// channel definition only if it is not already set, initializing the labels
// struct if needed.
func SetNotificationChannelIDIfAbsent(channel *NotificationChannelDefinition, id string) {
	if channel == nil {
		return
	}
	if channel.Metadata.Labels == nil {
		channel.Metadata.Labels = &NotificationChannelLabels{}
	}
	if channel.Metadata.Labels.Dash0Comid == nil {
		channel.Metadata.Labels.Dash0Comid = &id
	}
}

// SetNotificationChannelOrigin sets the dash0.com/origin label on a notification
// channel definition, initializing the labels struct if needed.
func SetNotificationChannelOrigin(channel *NotificationChannelDefinition, origin string) {
	if channel == nil {
		return
	}
	if channel.Metadata.Labels == nil {
		channel.Metadata.Labels = &NotificationChannelLabels{}
	}
	channel.Metadata.Labels.Dash0Comorigin = &origin
}
