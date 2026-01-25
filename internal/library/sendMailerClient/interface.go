package sendMailerClient

import "context"

// Client hides the SMTP and environment handling details from the caller.
type Client interface {
	Send(ctx context.Context, msg Message) error
}
