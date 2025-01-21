package echotask

import (
	"log/slog"

	"github.com/labstack/echo/v4"

	"github.com/zircuit-labs/zkr-go-common/calm"
	"github.com/zircuit-labs/zkr-go-common/log"
	"github.com/zircuit-labs/zkr-go-common/xerrors/errclass"
)

func Recover(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := calm.Unpanic(func() error {
				return next(c)
			})
			switch errclass.GetClass(err) {
			case errclass.Nil:
				return nil
			case errclass.Panic:
				logger.Error("middleware recovered from panic", log.ErrAttr(err))
				c.Error(err)
				return nil
			default:
				c.Error(err)
				return err
			}
		}
	}
}
