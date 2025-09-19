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

// HookChain allows chaining multiple hooks together
type HookChain struct {
	preHooks  []PreHook
	postHooks []PostHook
}

type HookChainOption func(*HookChain)

func WithPreHooksChain(hooks ...PreHook) HookChainOption {
	return func(hc *HookChain) {
		hc.preHooks = append(hc.preHooks, hooks...)
	}
}

func WithPostHooksChain(hooks ...PostHook) HookChainOption {
	return func(hc *HookChain) {
		hc.postHooks = append(hc.postHooks, hooks...)
	}
}

// NewHookChain creates a new hook chain
func NewHookChain(hooks ...HookChainOption) *HookChain {
	c := &HookChain{
		preHooks:  make([]PreHook, 0),
		postHooks: make([]PostHook, 0),
	}

	for _, h := range hooks {
		h(c)
	}

	return c
}

// AddPreHook adds a hook to the chain
func (c *HookChain) AddPreHook(hook PreHook) *HookChain {
	c.preHooks = append(c.preHooks, hook)
	return c
}

// AddPreHook adds a hook to the chain
func (c *HookChain) AddPostHook(hook PostHook) *HookChain {
	c.postHooks = append(c.postHooks, hook)
	return c
}

// Execute executes all hooks in the chain
func (c *HookChain) ExecutePreHooks(ctx *HookContext) error {
	for _, hook := range c.preHooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *HookChain) ExecutePostHooks(ctx *HookContext) (string, error) {

	response := ctx.Output
	for _, hook := range c.postHooks {
		if out, err := hook(ctx); err != nil {
			return "", err
		} else {
			response = out
			ctx.Output = out
		}
	}

	return response, nil
}

// AsPreHook returns the chain as a single PreHook
func (c *HookChain) AsPreHook() PreHook {
	return func(ctx *HookContext) error {
		return c.ExecutePreHooks(ctx)
	}
}

// AsPostHook returns the chain as a single PostHook
func (c *HookChain) AsPostHook() PostHook {
	return func(ctx *HookContext) (string, error) {
		return c.ExecutePostHooks(ctx)
	}
}
