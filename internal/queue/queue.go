package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"notify/internal/config"
	"notify/internal/service"
)

type Task struct {
	ID        string
	Channel   service.Channel
	To        string
	Message   any
	Attempts  int
	CreatedAt time.Time
}

type targetQueue struct {
	key     string
	tasks   chan *Task
	limiter *rate.Limiter
	svc     service.NotifyService
}

type Manager struct {
	queues     map[string]*targetQueue
	mu         sync.Mutex
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        config.QueueConfig
	taskSeq    uint64
	taskSeqMu  sync.Mutex
}

var manager *Manager

func Init(cfg config.QueueConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	manager = &Manager{
		queues: make(map[string]*targetQueue),
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}
}

func GetManager() *Manager {
	return manager
}

func (m *Manager) Enqueue(channel service.Channel, to string, message any) string {
	m.taskSeqMu.Lock()
	m.taskSeq++
	taskID := fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), m.taskSeq)
	m.taskSeqMu.Unlock()

	task := &Task{
		ID:        taskID,
		Channel:   channel,
		To:        to,
		Message:   message,
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	key := fmt.Sprintf("%s:%s", channel, to)

	m.mu.Lock()
	tq, exists := m.queues[key]
	if !exists {
		svc, err := service.GetService(channel)
		if err != nil {
			m.mu.Unlock()
			slog.Error("Failed to get service for queue", "channel", channel, "error", err)
			return taskID
		}

		tq = &targetQueue{
			key:     key,
			tasks:   make(chan *Task, 1000),
			limiter: rate.NewLimiter(rate.Limit(m.cfg.RatePerSecond), 1),
			svc:     svc,
		}
		m.queues[key] = tq

		m.wg.Add(1)
		go m.runWorker(tq)
	}
	m.mu.Unlock()

	select {
	case tq.tasks <- task:
		slog.Info("Task enqueued", "taskId", taskID, "channel", channel, "to", to)
	default:
		slog.Warn("Queue full, task dropped", "taskId", taskID, "channel", channel, "to", to)
	}

	return taskID
}

func (m *Manager) runWorker(tq *targetQueue) {
	defer m.wg.Done()

	idleTimer := time.NewTimer(5 * time.Minute)
	defer idleTimer.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.drainQueue(tq)
			return

		case task := <-tq.tasks:
			idleTimer.Reset(5 * time.Minute)
			m.processTask(tq, task)

		case <-idleTimer.C:
			m.mu.Lock()
			if len(tq.tasks) == 0 {
				delete(m.queues, tq.key)
				m.mu.Unlock()
				slog.Info("Queue idle, closing", "key", tq.key)
				return
			}
			m.mu.Unlock()
			idleTimer.Reset(5 * time.Minute)
		}
	}
}

func (m *Manager) processTask(tq *targetQueue, task *Task) {
	if err := tq.limiter.Wait(m.ctx); err != nil {
		return
	}

	for task.Attempts < m.cfg.MaxRetries {
		task.Attempts++
		_, err := tq.svc.SendRawMessage(task.To, task.Message)
		if err == nil {
			slog.Info("Message sent", "taskId", task.ID, "attempt", task.Attempts)
			return
		}
		slog.Warn("Send failed", "taskId", task.ID, "attempt", task.Attempts, "error", err)

		if task.Attempts < m.cfg.MaxRetries {
			time.Sleep(m.cfg.RetryDelay)
		}
	}
	slog.Error("Send failed after retries", "taskId", task.ID, "attempts", m.cfg.MaxRetries)
}

func (m *Manager) drainQueue(tq *targetQueue) {
	for {
		select {
		case task := <-tq.tasks:
			m.processTask(tq, task)
		default:
			return
		}
	}
}

func (m *Manager) Shutdown() {
	slog.Info("Shutting down queue manager...")
	m.cancel()
	m.wg.Wait()
	slog.Info("Queue manager shutdown complete")
}
