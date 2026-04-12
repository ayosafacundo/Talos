package hub

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	hubpb "Talos/api/proto/talos/hub/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPipeEndpoint = `\\.\pipe\talos_hub`
)

// Handler is a local app request handler used by host-managed apps.
type Handler func(ctx context.Context, msg *hubpb.Message) (*hubpb.Message, error)

// Server hosts the central gRPC message router.
type Server struct {
	mu        sync.RWMutex
	handlers  map[string]Handler
	socketURL string

	stateSaveHook         func(appID string, data []byte) error
	stateLoadHook         func(appID string) ([]byte, bool, error)
	permissionRequestHook func(appID, scope, reason string) (bool, string, error)
	resolvePathHook       func(appID, relativePath string) (string, bool, error)

	grpcServer *grpc.Server
	listener   net.Listener

	hubpb.UnimplementedHubServiceServer
}

// NewServer creates a new unstarted hub server.
func NewServer(socketURL string) *Server {
	return &Server{
		handlers:  make(map[string]Handler),
		socketURL: socketURL,
	}
}

// SocketURL returns the current transport endpoint.
func (s *Server) SocketURL() string {
	return s.socketURL
}

// Start binds the transport and starts serving.
func (s *Server) Start() error {
	if s.socketURL == "" {
		s.socketURL = DefaultSocketURL()
	}

	listener, err := listenLocal(s.socketURL)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	hubpb.RegisterHubServiceServer(server, s)

	s.mu.Lock()
	s.grpcServer = server
	s.listener = listener
	s.mu.Unlock()

	go server.Serve(listener)
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		s.grpcServer = nil
	}
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
	if runtime.GOOS != "windows" && strings.HasPrefix(s.socketURL, "unix://") {
		_ = os.Remove(strings.TrimPrefix(s.socketURL, "unix://"))
	}
}

// RegisterHandler registers a local app route target.
func (s *Server) RegisterHandler(appID string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[appID] = handler
}

// UnregisterHandler removes a local app route target.
func (s *Server) UnregisterHandler(appID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.handlers, appID)
}

func (s *Server) SetStateHooks(
	save func(appID string, data []byte) error,
	load func(appID string) ([]byte, bool, error),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateSaveHook = save
	s.stateLoadHook = load
}

func (s *Server) SetPermissionRequestHook(
	hook func(appID, scope, reason string) (bool, string, error),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissionRequestHook = hook
}

func (s *Server) SetResolvePathHook(
	hook func(appID, relativePath string) (string, bool, error),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolvePathHook = hook
}

func (s *Server) Route(ctx context.Context, req *hubpb.RouteRequest) (*hubpb.RouteResponse, error) {
	if req == nil || req.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	msg := req.Message
	if strings.TrimSpace(msg.TargetAppId) == "" {
		return nil, status.Error(codes.InvalidArgument, "target_app_id is required")
	}

	s.mu.RLock()
	handler := s.handlers[msg.TargetAppId]
	s.mu.RUnlock()
	if handler == nil {
		return nil, status.Errorf(codes.NotFound, "target handler %q not found", msg.TargetAppId)
	}

	response, err := handler(ctx, msg)
	if err != nil {
		return &hubpb.RouteResponse{Error: err.Error()}, nil
	}

	return &hubpb.RouteResponse{Message: response}, nil
}

func (s *Server) Broadcast(ctx context.Context, req *hubpb.BroadcastRequest) (*hubpb.BroadcastResponse, error) {
	if req == nil || req.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	sourceID := req.Message.SourceAppId
	s.mu.RLock()
	handlers := make([]Handler, 0, len(s.handlers))
	for appID, h := range s.handlers {
		if appID == sourceID {
			continue
		}
		handlers = append(handlers, h)
	}
	s.mu.RUnlock()

	var sent int32
	for _, handler := range handlers {
		if _, err := handler(ctx, req.Message); err == nil {
			sent++
		}
	}

	return &hubpb.BroadcastResponse{RecipientCount: sent}, nil
}

func (s *Server) SaveState(_ context.Context, req *hubpb.SaveStateRequest) (*hubpb.SaveStateResponse, error) {
	if req == nil || strings.TrimSpace(req.AppId) == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	s.mu.RLock()
	save := s.stateSaveHook
	s.mu.RUnlock()
	if save == nil {
		return &hubpb.SaveStateResponse{Ok: false, Error: "state store is not configured"}, nil
	}

	if err := save(req.AppId, req.Data); err != nil {
		return &hubpb.SaveStateResponse{Ok: false, Error: err.Error()}, nil
	}
	return &hubpb.SaveStateResponse{Ok: true}, nil
}

func (s *Server) LoadState(_ context.Context, req *hubpb.LoadStateRequest) (*hubpb.LoadStateResponse, error) {
	if req == nil || strings.TrimSpace(req.AppId) == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	s.mu.RLock()
	load := s.stateLoadHook
	s.mu.RUnlock()
	if load == nil {
		return &hubpb.LoadStateResponse{Found: false, Error: "state store is not configured"}, nil
	}

	data, found, err := load(req.AppId)
	if err != nil {
		return &hubpb.LoadStateResponse{Found: false, Error: err.Error()}, nil
	}

	return &hubpb.LoadStateResponse{Data: data, Found: found}, nil
}

func (s *Server) RequestPermission(_ context.Context, req *hubpb.PermissionRequest) (*hubpb.PermissionResponse, error) {
	if req == nil || strings.TrimSpace(req.AppId) == "" || strings.TrimSpace(req.Scope) == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and scope are required")
	}

	s.mu.RLock()
	hook := s.permissionRequestHook
	s.mu.RUnlock()
	if hook == nil {
		return &hubpb.PermissionResponse{
			Granted: false,
			Message: "permission hook is not configured",
		}, nil
	}

	granted, message, err := hook(req.AppId, req.Scope, req.Reason)
	if err != nil {
		return &hubpb.PermissionResponse{Granted: false, Message: err.Error()}, nil
	}

	return &hubpb.PermissionResponse{Granted: granted, Message: message}, nil
}

func (s *Server) ResolvePath(_ context.Context, req *hubpb.ResolvePathRequest) (*hubpb.ResolvePathResponse, error) {
	if req == nil || strings.TrimSpace(req.AppId) == "" || strings.TrimSpace(req.RelativePath) == "" {
		return nil, status.Error(codes.InvalidArgument, "app_id and relative_path are required")
	}

	s.mu.RLock()
	hook := s.resolvePathHook
	s.mu.RUnlock()
	if hook == nil {
		return &hubpb.ResolvePathResponse{
			Allowed: false,
			Error:   "resolve path hook is not configured",
		}, nil
	}

	resolved, allowed, err := hook(req.AppId, req.RelativePath)
	if err != nil {
		return &hubpb.ResolvePathResponse{Allowed: false, Error: err.Error()}, nil
	}
	return &hubpb.ResolvePathResponse{
		ResolvedPath: resolved,
		Allowed:      allowed,
	}, nil
}

// DefaultSocketURL returns the platform socket endpoint for the hub.
func DefaultSocketURL() string {
	if runtime.GOOS == "windows" {
		return "npipe://" + defaultPipeEndpoint
	}
	return "unix://" + filepath.Join(os.TempDir(), "talos_hub.sock")
}
