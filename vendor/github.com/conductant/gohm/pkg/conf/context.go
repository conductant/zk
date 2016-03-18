package conf

import (
	"github.com/conductant/gohm/pkg/encoding"
	"golang.org/x/net/context"
)

type initialDataContextKey int
type configDataTypeContextKey int

var (
	InitialDataContextKey    initialDataContextKey    = 1
	ConfigDataTypeContextKey configDataTypeContextKey = 2
)

func ContextPutInitialData(ctx context.Context, data interface{}) context.Context {
	return context.WithValue(ctx, InitialDataContextKey, data)
}
func ContextGetInitialData(ctx context.Context) interface{} {
	return ctx.Value(InitialDataContextKey)
}
func ContextPutConfigDataType(ctx context.Context, t encoding.ContentType) context.Context {
	return context.WithValue(ctx, ConfigDataTypeContextKey, t)
}
func ContextGetConfigDataType(ctx context.Context) encoding.ContentType {
	if v, ok := ctx.Value(ConfigDataTypeContextKey).(encoding.ContentType); ok {
		return v
	}
	return encoding.ContentTypeDefault
}
