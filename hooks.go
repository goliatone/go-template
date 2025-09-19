package template

import (
	"sort"
	"sync"
)

// HookContext provides context for generation hooks
type HookContext struct {
	TemplateName string
	Template     string
	Data         any
	Output       string
	Metadata     map[string]any
	IsPreHook    bool
}

// GenerationHook is a function that can modify template data or output
type GenerationHook func(ctx *HookContext) error

type PreHook func(ctx *HookContext) error // modify Data or Metadata
type PostHook func(ctx *HookContext) (string, error)

// HookCondition allows callers to decide whether a hook should run for a given context.
type HookCondition func(ctx *HookContext) bool

type HookManager struct {
	mu        sync.RWMutex
	preHooks  map[int][]PreHook
	postHooks map[int][]PostHook
}

func NewHooksManager() *HookManager {
	return &HookManager{
		preHooks:  make(map[int][]PreHook, 0),
		postHooks: make(map[int][]PostHook, 0),
	}
}

// AddPreHook registers a pre generation hook
func (e *HookManager) AddPreHook(hook PreHook, priority ...int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}

	hooks, ok := e.preHooks[p]
	if !ok {
		hooks = make([]PreHook, 0)
	}

	e.preHooks[p] = append(hooks, hook)

}

// AddPostHook registers a post generation hook
func (e *HookManager) AddPostHook(hook PostHook, priority ...int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}

	hooks, ok := e.postHooks[p]
	if !ok {
		hooks = make([]PostHook, 0)
	}

	e.postHooks[p] = append(hooks, hook)
}

func (e *HookManager) PreHooks() []PreHook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keys := []int{}
	for k := range e.preHooks {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	out := make([]PreHook, 0)
	for _, priority := range keys {
		out = append(out, e.preHooks[priority]...)
	}

	return out
}

func (e *HookManager) PostHooks() []PostHook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keys := []int{}
	for k := range e.postHooks {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	out := make([]PostHook, 0)
	for _, priority := range keys {
		out = append(out, e.postHooks[priority]...)
	}

	return out
}
