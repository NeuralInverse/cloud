package messagepartbuffer

import (
	"container/heap"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/quartz"
)

const (
	defaultMaxEpisodeBytes        = int64(1024 * 1024)
	defaultClosedEpisodeRetention = 15 * time.Second
	defaultSubscriberSendTimeout  = 10 * time.Second
	defaultSubscriberChannelSize  = 16
)

var (
	// ErrEpisodeExists means the episode already exists.
	ErrEpisodeExists = xerrors.New("message part episode already exists")
	// ErrEpisodeNotFound means the episode has not been created.
	ErrEpisodeNotFound = xerrors.New("message part episode not found")
	// ErrEpisodeClosed means the episode no longer accepts parts.
	ErrEpisodeClosed = xerrors.New("message part episode closed")
	// ErrEpisodeFull means the episode byte limit would be exceeded.
	ErrEpisodeFull = xerrors.New("message part episode full")
	// ErrMessagePartBufferClosed means the whole buffer is closed.
	ErrMessagePartBufferClosed = xerrors.New("message part buffer closed")
)

// Key identifies a buffered message part episode.
type Key struct {
	ChatID            uuid.UUID
	HistoryVersion    int64
	GenerationAttempt int64
}

// Part is a buffered chat message part with its sequence number.
type Part struct {
	Seq         int64
	Role        codersdk.ChatMessageRole
	MessagePart codersdk.ChatMessagePart
}

type partJSON struct {
	Seq  int64                    `json:"seq"`
	Role codersdk.ChatMessageRole `json:"role"`
	Part codersdk.ChatMessagePart `json:"part"`
}

func (p Part) jsonValue() partJSON {
	return partJSON{
		Seq:  p.Seq,
		Role: p.Role,
		Part: p.MessagePart,
	}
}

// Options configures a Buffer.
type Options struct {
	MaxEpisodeBytes        int64
	ClosedEpisodeRetention time.Duration
	SubscriberSendTimeout  time.Duration
	SubscriberChannelSize  int
	Clock                  quartz.Clock
}

// Buffer stores streamed message parts by episode.
type Buffer struct {
	mu             sync.Mutex
	opts           Options
	episodes       map[Key]*episodeState
	closedEpisodes closedEpisodeHeap
	closed         bool
	done           chan struct{}
}

type episodeState struct {
	created        bool
	createdCh      chan struct{}
	waiters        int
	closed         bool
	closedAt       time.Time
	closedHeapItem *closedEpisodeItem
	parts          []Part
	bytes          int64
	subscribers    map[*episodeSubscriber]struct{}
}

type closedEpisodeItem struct {
	key      Key
	closedAt time.Time
}

type closedEpisodeHeap []*closedEpisodeItem

func (h closedEpisodeHeap) Len() int {
	return len(h)
}

func (h closedEpisodeHeap) Less(i, j int) bool {
	return h[i].closedAt.Before(h[j].closedAt)
}

func (h closedEpisodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *closedEpisodeHeap) Push(value any) {
	item, ok := value.(*closedEpisodeItem)
	if !ok {
		panic("closed episode heap received invalid item")
	}
	*h = append(*h, item)
}

func (h *closedEpisodeHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	old[len(old)-1] = nil
	*h = old[:len(old)-1]
	return last
}

type episodeSubscriber struct {
	out       chan Part
	queue     chan Part
	stopCh    chan struct{}
	queueOnce sync.Once
	stopOnce  sync.Once
}

// New returns a message part buffer.
func New(options Options) *Buffer {
	if options.MaxEpisodeBytes <= 0 {
		options.MaxEpisodeBytes = defaultMaxEpisodeBytes
	}
	if options.ClosedEpisodeRetention <= 0 {
		options.ClosedEpisodeRetention = defaultClosedEpisodeRetention
	}
	if options.SubscriberSendTimeout <= 0 {
		options.SubscriberSendTimeout = defaultSubscriberSendTimeout
	}
	if options.SubscriberChannelSize < 0 {
		options.SubscriberChannelSize = 0
	}
	if options.SubscriberChannelSize == 0 {
		options.SubscriberChannelSize = defaultSubscriberChannelSize
	}
	if options.Clock == nil {
		options.Clock = quartz.NewReal()
	}
	return &Buffer{
		opts:     options,
		episodes: make(map[Key]*episodeState),
		done:     make(chan struct{}),
	}
}

// CreateEpisode creates a new episode.
func (b *Buffer) CreateEpisode(key Key) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrMessagePartBufferClosed
	}
	b.gcClosedEpisodesLocked(b.opts.Clock.Now("message-part-buffer", "create"))
	episode := b.episodeLocked(key)
	if episode.created {
		return ErrEpisodeExists
	}
	episode.created = true
	close(episode.createdCh)
	return nil
}

// AddPart appends a part to an existing episode.
func (b *Buffer) AddPart(key Key, role codersdk.ChatMessageRole, part codersdk.ChatMessagePart) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrMessagePartBufferClosed
	}
	episode := b.episodes[key]
	if episode == nil || !episode.created {
		return ErrEpisodeNotFound
	}
	if episode.closed {
		return ErrEpisodeClosed
	}
	buffered := Part{
		Seq:         int64(len(episode.parts) + 1),
		Role:        role,
		MessagePart: part,
	}
	sizeBytes, err := serializedPartBytes(buffered)
	if err != nil {
		return err
	}
	if episode.bytes+sizeBytes > b.opts.MaxEpisodeBytes {
		return ErrEpisodeFull
	}
	episode.parts = append(episode.parts, buffered)
	episode.bytes += sizeBytes
	for subscriber := range episode.subscribers {
		select {
		case subscriber.queue <- buffered:
		default:
			b.stopSubscriberLocked(episode, subscriber)
		}
	}
	return nil
}

// CloseEpisode marks an episode closed and closes its subscribers.
func (b *Buffer) CloseEpisode(key Key) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrMessagePartBufferClosed
	}
	episode := b.episodeLocked(key)
	if !episode.created {
		episode.created = true
		close(episode.createdCh)
	}
	if episode.closed {
		return nil
	}
	episode.closed = true
	episode.closedAt = b.opts.Clock.Now("message-part-buffer", "close")
	b.queueClosedEpisodeLocked(key, episode)
	for subscriber := range episode.subscribers {
		subscriber.closeQueue()
	}
	return nil
}

// GetParts returns a snapshot of buffered parts for an episode.
func (b *Buffer) GetParts(key Key) ([]Part, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrMessagePartBufferClosed
	}
	b.gcClosedEpisodesLocked(b.opts.Clock.Now("message-part-buffer", "get"))
	episode := b.episodes[key]
	if episode == nil || !episode.created {
		return nil, ErrEpisodeNotFound
	}
	return append([]Part(nil), episode.parts...), nil
}

// SubscribeToEpisode replays existing parts and streams new parts.
func (b *Buffer) SubscribeToEpisode(ctx context.Context, key Key) (<-chan Part, func(), error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, nil, ErrMessagePartBufferClosed
	}
	episode := b.episodeLocked(key)
	createdCh := episode.createdCh
	created := episode.created
	if !created {
		episode.waiters++
	}
	b.mu.Unlock()

	if !created {
		select {
		case <-createdCh:
			b.finishEpisodeWait(key, episode)
		case <-ctx.Done():
			b.finishEpisodeWait(key, episode)
			return nil, nil, ctx.Err()
		case <-b.done:
			b.finishEpisodeWait(key, episode)
			return nil, nil, ErrMessagePartBufferClosed
		}
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, nil, ErrMessagePartBufferClosed
	}
	episode = b.episodes[key]
	if episode == nil || !episode.created {
		b.mu.Unlock()
		return nil, nil, ErrEpisodeNotFound
	}
	subscriber := &episodeSubscriber{
		out:    make(chan Part),
		queue:  make(chan Part, b.opts.SubscriberChannelSize),
		stopCh: make(chan struct{}),
	}
	if episode.subscribers == nil {
		episode.subscribers = make(map[*episodeSubscriber]struct{})
	}
	episode.subscribers[subscriber] = struct{}{}
	initial := append([]Part(nil), episode.parts...)
	closed := episode.closed
	if closed {
		subscriber.closeQueue()
	}
	b.mu.Unlock()

	go b.deliverSubscriber(ctx, key, subscriber, initial)
	cancel := func() {
		b.cancelSubscriber(key, subscriber)
	}
	return subscriber.out, cancel, nil
}

// Close closes the buffer and all pending subscriptions.
func (b *Buffer) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	close(b.done)
	for _, episode := range b.episodes {
		for subscriber := range episode.subscribers {
			b.stopSubscriberLocked(episode, subscriber)
		}
		if !episode.created {
			episode.created = true
			close(episode.createdCh)
		}
	}
	b.mu.Unlock()
}

func (b *Buffer) gcClosedEpisodesLocked(now time.Time) {
	cutoff := now.Add(-b.opts.ClosedEpisodeRetention)
	type retainedEpisode struct {
		key     Key
		episode *episodeState
	}
	retained := make([]retainedEpisode, 0)
	for b.closedEpisodes.Len() > 0 {
		item := b.closedEpisodes[0]
		if item.closedAt.After(cutoff) {
			break
		}
		popped, ok := heap.Pop(&b.closedEpisodes).(*closedEpisodeItem)
		if !ok || popped != item {
			continue
		}
		episode := b.episodes[item.key]
		if episode == nil || episode.closedHeapItem != item || !episode.closed {
			continue
		}
		episode.closedHeapItem = nil
		if len(episode.subscribers) > 0 {
			retained = append(retained, retainedEpisode{key: item.key, episode: episode})
			continue
		}
		delete(b.episodes, item.key)
	}
	for _, item := range retained {
		if b.episodes[item.key] != item.episode || !item.episode.closed || item.episode.closedHeapItem != nil {
			continue
		}
		b.queueClosedEpisodeLocked(item.key, item.episode)
	}
}

func (b *Buffer) queueClosedEpisodeLocked(key Key, episode *episodeState) {
	if episode.closedHeapItem != nil {
		return
	}
	item := &closedEpisodeItem{key: key, closedAt: episode.closedAt}
	episode.closedHeapItem = item
	heap.Push(&b.closedEpisodes, item)
}

func (b *Buffer) episodeLocked(key Key) *episodeState {
	episode := b.episodes[key]
	if episode != nil {
		return episode
	}
	episode = &episodeState{createdCh: make(chan struct{})}
	b.episodes[key] = episode
	return episode
}

func (b *Buffer) finishEpisodeWait(key Key, waited *episodeState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	episode := b.episodes[key]
	if episode != waited {
		return
	}
	if episode.waiters > 0 {
		episode.waiters--
	}
	if !episode.created && episode.waiters == 0 && len(episode.subscribers) == 0 {
		delete(b.episodes, key)
	}
}

func (b *Buffer) deliverSubscriber(ctx context.Context, key Key, subscriber *episodeSubscriber, initial []Part) {
	defer close(subscriber.out)
	defer b.removeSubscriber(key, subscriber)
	for _, part := range initial {
		if !b.sendSubscriberPart(ctx, subscriber, part) {
			return
		}
	}
	for {
		select {
		case part, ok := <-subscriber.queue:
			if !ok {
				return
			}
			if !b.sendSubscriberPart(ctx, subscriber, part) {
				return
			}
		case <-subscriber.stopCh:
			return
		case <-ctx.Done():
			return
		case <-b.done:
			return
		}
	}
}

func (b *Buffer) sendSubscriberPart(ctx context.Context, subscriber *episodeSubscriber, part Part) bool {
	timer := b.opts.Clock.NewTimer(b.opts.SubscriberSendTimeout, "message-part-buffer", "subscriber-send")
	defer timer.Stop()
	select {
	case subscriber.out <- part:
		return true
	case <-timer.C:
		return false
	case <-subscriber.stopCh:
		return false
	case <-ctx.Done():
		return false
	case <-b.done:
		return false
	}
}

func (b *Buffer) cancelSubscriber(key Key, subscriber *episodeSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	episode := b.episodes[key]
	if episode != nil {
		b.stopSubscriberLocked(episode, subscriber)
		return
	}
	subscriber.stop()
	subscriber.closeQueue()
}

func (b *Buffer) removeSubscriber(key Key, subscriber *episodeSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	episode := b.episodes[key]
	if episode == nil {
		return
	}
	delete(episode.subscribers, subscriber)
	if episode.closed && len(episode.subscribers) == 0 {
		b.queueClosedEpisodeLocked(key, episode)
	}
	subscriber.closeQueue()
}

func (*Buffer) stopSubscriberLocked(episode *episodeState, subscriber *episodeSubscriber) {
	delete(episode.subscribers, subscriber)
	subscriber.stop()
	subscriber.closeQueue()
}

func (s *episodeSubscriber) closeQueue() {
	s.queueOnce.Do(func() { close(s.queue) })
}

func (s *episodeSubscriber) stop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
}

func serializedPartBytes(part Part) (int64, error) {
	data, err := json.Marshal(part.jsonValue())
	if err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}
