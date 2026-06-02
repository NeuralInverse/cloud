package chatcompletions

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type sseStreamStats struct {
	responseReceived atomic.Bool
	statusCode       atomic.Int64
	contentType      atomic.Value

	rawBytes    atomic.Int64
	dataEvents  atomic.Int64
	comments    atomic.Int64
	emptyEvents atomic.Int64
	sawDone     atomic.Bool
}

func (s *sseStreamStats) wrapResponse(resp *http.Response) {
	if resp == nil {
		return
	}
	s.responseReceived.Store(true)
	s.statusCode.Store(int64(resp.StatusCode))
	s.contentType.Store(resp.Header.Get("Content-Type"))

	if resp.Body == nil || !s.isSSEUpstream() {
		return
	}
	resp.Body = newSSEFilteringBody(resp.Body, s)
}

func (s *sseStreamStats) isSSEUpstream() bool {
	if !s.responseReceived.Load() {
		return false
	}
	statusCode := int(s.statusCode.Load())
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return false
	}
	return isEventStreamContentType(s.contentTypeString())
}

func (s *sseStreamStats) contentTypeString() string {
	contentType, _ := s.contentType.Load().(string)
	return contentType
}

func (s *sseStreamStats) statusCodeInt() int {
	return int(s.statusCode.Load())
}

func (s *sseStreamStats) hasDataEvents() bool {
	return s != nil && s.dataEvents.Load() > 0
}

func (s *sseStreamStats) isEmptyDataStream() bool {
	return s != nil && s.isSSEUpstream() && s.dataEvents.Load() == 0 && !s.sawDone.Load()
}

func isEventStreamContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	return strings.EqualFold(mediaType, "text/event-stream")
}

type sseFilteringBody struct {
	reader   *io.PipeReader
	upstream io.ReadCloser

	closeOnce sync.Once
	closeErr  error
}

func newSSEFilteringBody(upstream io.ReadCloser, stats *sseStreamStats) io.ReadCloser {
	reader, writer := io.Pipe()
	body := &sseFilteringBody{
		reader:   reader,
		upstream: upstream,
	}
	go body.filter(writer, stats)
	return body
}

func (b *sseFilteringBody) Read(p []byte) (int, error) {
	return b.reader.Read(p)
}

func (b *sseFilteringBody) Close() error {
	return errors.Join(b.reader.Close(), b.closeUpstream())
}

func (b *sseFilteringBody) closeUpstream() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.upstream.Close()
	})
	return b.closeErr
}

func (b *sseFilteringBody) filter(writer *io.PipeWriter, stats *sseStreamStats) {
	defer func() {
		_ = b.closeUpstream()
	}()

	scanner := bufio.NewScanner(b.upstream)
	scanner.Buffer(nil, bufio.MaxScanTokenSize<<9)

	var event bytes.Buffer
	var eventData bytes.Buffer
	var eventHasData bool
	var eventHadLine bool

	resetEvent := func() {
		event.Reset()
		eventData.Reset()
		eventHasData = false
		eventHadLine = false
	}
	dispatchEvent := func() error {
		defer resetEvent()
		if !eventHasData {
			if eventHadLine {
				stats.emptyEvents.Add(1)
			}
			return nil
		}

		if bytes.HasPrefix(eventData.Bytes(), []byte("[DONE]")) {
			stats.sawDone.Store(true)
		} else {
			stats.dataEvents.Add(1)
		}

		if _, err := writer.Write(event.Bytes()); err != nil {
			return err
		}
		_, err := writer.Write([]byte("\n"))
		return err
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		stats.rawBytes.Add(int64(len(line) + 1))

		if len(line) == 0 {
			if err := dispatchEvent(); err != nil {
				_ = writer.CloseWithError(err)
				return
			}
			continue
		}

		eventHadLine = true
		name, value, _ := bytes.Cut(line, []byte(":"))
		if len(name) == 0 {
			stats.comments.Add(1)
			continue
		}
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}
		if bytes.Equal(name, []byte("data")) {
			eventHasData = true
			_, _ = eventData.Write(value)
			_, _ = eventData.WriteRune('\n')
		}

		_, _ = event.Write(line)
		_ = event.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		_ = writer.CloseWithError(err)
		return
	}
	if eventHadLine {
		if err := dispatchEvent(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
	}
	_ = writer.Close()
}
