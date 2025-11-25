package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"
)

func TestGetHealthCheck_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Parallel()

	tests := []struct {
		name       string
		setup      func(context.Context, *MockChecker)
		wantErr    bool
		wantStatus int
	}{
		{
			name: "Ping fails",
			setup: func(ctx context.Context, mockChecker *MockChecker) {
				mockChecker.EXPECT().HealthCheck(ctx).Return(errors.New("can't ping the database"))
			},
			wantErr:    false,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Ping works",
			setup: func(ctx context.Context, mockChecker *MockChecker) {
				mockChecker.EXPECT().HealthCheck(ctx).Return(nil)
			},
			wantErr:    false,
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockChecker := NewMockChecker(ctrl)

			e := echo.New()

			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			rec := httptest.NewRecorder()
			echoContext := e.NewContext(req, rec)

			tt.setup(req.Context(), mockChecker)

			g := New(mockChecker)
			if err := g.Handle(echoContext); (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantStatus != rec.Code {
				t.Errorf("Status Code = %d, wantStatus %d", rec.Code, tt.wantStatus)
			}
		})
	}

	ctrl.Finish()
}
