package ctrl

import (
	"context"
	"sync"
	"time"

	"github.com/thanhminhmr/go-common/log"

	"github.com/thanhminhmr/go-exception"
)

type Starter = func(ctx context.Context) (Runner, Cleaner, error)
type Runner = func(ctx context.Context, shutdown context.CancelFunc)
type Cleaner = func(ctx context.Context)

type Config struct {
	StarterTimeout uint `env:"CONTROLLER_STARTER_TIMEOUT" default:"30"`
	CleanerTimeout uint `env:"CONTROLLER_CLEANER_TIMEOUT" default:"30"`
}

func New(config *Config) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		config: config,
		ctx:    log.IntoCtx(ctx),
		cancel: cancel,
	}
}

type Controller struct {
	config   *Config
	ctx      *log.Ctx
	cancel   context.CancelFunc
	running  sync.WaitGroup
	cleaning sync.WaitGroup
	runners  []Runner
}

func (c *Controller) prepare(starter Starter) (runner Runner, cleaner Cleaner, err error) {
	defer func() {
		if err != nil {
			c.cancel()
			c.cleaning.Wait()
		}
	}()
	defer exception.Recover(func(recovered exception.Exception) {
		c.ctx.Error("Starter panicked", "recovered", recovered)
		err = recovered
	})
	ctx := context.Context(c.ctx)
	if c.config.StarterTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(c.ctx, time.Duration(c.config.StarterTimeout)*time.Second)
		defer cancel()
	}
	return starter(ctx)
}

func (c *Controller) Add(starter Starter) error {
	// prepare runner
	runner, cleaner, err := c.prepare(starter)
	if err != nil {
		return err
	}
	// register cleaner
	if cleaner != nil {
		c.cleaning.Add(1)
		context.AfterFunc(c.ctx, func() {
			defer c.cleaning.Done()
			defer exception.Recover(func(recovered exception.Exception) {
				c.ctx.Error("Cleaner panicked", "recovered", recovered)
			})
			ctx := context.Context(c.ctx)
			if c.config.CleanerTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(c.ctx, time.Duration(c.config.CleanerTimeout)*time.Second)
				defer cancel()
			}
			cleaner(ctx)
		})
	}
	// register runner
	c.runners = append(c.runners, runner)
	return nil
}

func (c *Controller) Run() {
	if c.ctx.Err() != nil {
		panic("BUG: controller is already dead")
	}
	for _, runner := range c.runners {
		// start the runner
		c.running.Add(1)
		go func() {
			defer c.running.Done()
			defer exception.Recover(func(recovered exception.Exception) {
				c.ctx.Error("Runner panicked", "recovered", recovered)
			})
			runner(c.ctx, c.cancel)
		}()
	}
	// wait for all runner/clean up to finish
	c.running.Wait()
	if c.ctx.Err() == nil {
		c.cancel()
	}
	c.cleaning.Done()
}
