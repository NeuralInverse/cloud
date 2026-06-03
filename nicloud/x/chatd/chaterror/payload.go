package chaterror

import (
	"time"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TerminalErrorPayload(classified ClassifiedError) *nicloudsdk.ChatError {
	if classified.Message == "" {
		return nil
	}
	return &nicloudsdk.ChatError{
		Message:    classified.Message,
		Detail:     classified.Detail,
		Kind:       classified.Kind,
		Provider:   classified.Provider,
		Retryable:  classified.Retryable,
		StatusCode: classified.StatusCode,
	}
}

func StreamRetryPayload(
	attempt int,
	delay time.Duration,
	classified ClassifiedError,
) *nicloudsdk.ChatStreamRetry {
	if classified.Message == "" {
		return nil
	}
	return &nicloudsdk.ChatStreamRetry{
		Attempt:    attempt,
		DelayMs:    delay.Milliseconds(),
		Error:      retryMessage(classified),
		Kind:       classified.Kind,
		Provider:   classified.Provider,
		StatusCode: classified.StatusCode,
		RetryingAt: time.Now().Add(delay),
	}
}
