package errnie

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func (ambient AmbientContext) Get(UUID uuid.UUID) (
	map[ContextType]context.Context, map[ContextType]context.CancelFunc,
) {
	if ctx := ambient.ctxs[UUID]; len(ctx) == 0 {
		return nil, nil
	}

	return ambient.ctxs[UUID], ambient.cnls[UUID]
}

func (ambient AmbientContext) Set(
	UUID uuid.UUID, bg, cn, to, dl bool,
) (context.Context, []context.CancelFunc) {
	var ctx context.Context
	var cnl context.CancelFunc
	var cnls []context.CancelFunc

	ctx = context.TODO()

	if bg {
		ctx = context.Background()
	}

	if cn {
		ctx, cnl = context.WithCancel(ctx)
		cnls = append(cnls, cnl)
	}

	if to {
		ctx, cnl = context.WithTimeout(ctx, viper.GetDuration("errnie.contexts.default.timeout")*time.Second)
		cnls = append(cnls, cnl)
	}

	if dl {
		ctx, cnl = context.WithDeadline(ctx, viper.GetTime("errnie.contexts.default.deadline"))
		cnls = append(cnls, cnl)
	}

	return ctx, cnls
}
