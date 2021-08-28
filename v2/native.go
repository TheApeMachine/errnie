package errnie

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func (ambient AmbientContext) Cancel(UUID uuid.UUID) {
	defer delete(ambient.ctxs, UUID)
	defer delete(ambient.cnls, UUID)

	for _, cnl := range ambient.cnls[UUID] {
		cnl()
	}
}

func (ambient AmbientContext) Get(UUID uuid.UUID) ([]context.Context, []context.CancelFunc) {
	return ambient.ctxs[UUID], ambient.cnls[UUID]
}

func (ambient AmbientContext) Set(bg, cn, to, dl bool) uuid.UUID {
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

	uuid := uuid.New()
	ambient.ctxs[uuid] = append(ambient.ctxs[uuid], ctx)
	ambient.cnls[uuid] = cnls

	return uuid
}
