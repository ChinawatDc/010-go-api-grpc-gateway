# go-api-grpc-gateway — ใช้ grpc-gateway เพื่อเชื่อม gRPC กับ HTTP (Go) แบบละเอียด

บทนี้ทำให้คุณ **นิยาม API ครั้งเดียวใน `.proto`** แล้วได้ทั้ง:
- gRPC Server (สำหรับ internal / microservices)
- HTTP/JSON REST endpoint ผ่าน **grpc-gateway** (สำหรับ web/mobile/3rd-party)
- (Optional) Swagger/OpenAPI spec ผ่าน `protoc-gen-openapiv2`

> ต่างจากบทก่อนที่ REST (Gin) เป็น adapter เรียก gRPC client เอง  
> บทนี้คือ “generate HTTP gateway” จาก proto แบบมาตรฐาน production

---

## 0) Prerequisites

### 0.1 protoc
ตรวจสอบ:
```bash
protoc --version
```

Ubuntu/Debian/WSL:
```bash
sudo apt update
sudo apt install -y protobuf-compiler
```

### 0.2 Go plugins ที่ต้องใช้
ติดตั้ง (ครั้งเดียวต่อเครื่อง):
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

เพิ่ม PATH (ถ้า Windows/WSL หรือหา plugin ไม่เจอ):
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

ตรวจสอบ:
```bash
protoc-gen-go --version
protoc-gen-go-grpc --version
protoc-gen-grpc-gateway --version
protoc-gen-openapiv2 --version
```

---

## 1) Create Project Structure

```bash
mkdir -p go-api-grpc-gateway
cd go-api-grpc-gateway

mkdir -p proto/user/v1
mkdir -p gen/go
mkdir -p third_party
mkdir -p internal/user
mkdir -p internal/transport/grpcserver
mkdir -p internal/transport/gateway
mkdir -p cmd/grpc-server
mkdir -p cmd/gateway
mkdir -p openapi
```

โครงสร้าง:
```text
go-api-grpc-gateway/
  proto/
    user/
      v1/
        user.proto
  third_party/
    google/
      api/
        annotations.proto
        http.proto
  gen/
    go/
      (generated: pb.go, grpc.pb.go, gw.pb.go)
  internal/
    user/
      service.go
      store.go
    transport/
      grpcserver/
        handler.go
        server.go
      gateway/
        gateway.go
  cmd/
    grpc-server/
      main.go
    gateway/
      main.go
  openapi/
    (swagger json จะมาอยู่ตรงนี้)
  Makefile
  go.mod
  README.md
```

---

## 2) Initialize Go Module

> ปรับ module path ให้ตรง repo จริงของคุณ

```bash
go mod init github.com/ChinawatDc/go-api-grpc-gateway
```

deps:
```bash
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go get github.com/grpc-ecosystem/grpc-gateway/v2@latest
go get github.com/google/uuid@latest
```

---

## 3) Prepare google.api annotations (สำคัญมาก)

grpc-gateway ต้องใช้ไฟล์ proto ของ `google/api/http.proto` และ `annotations.proto`

### วิธีที่ง่ายและตรง (แนะนำสำหรับคอร์ส)
ให้ copy proto มาไว้ใน repo ภายใต้ `third_party/google/api/`

โครงสร้างที่ต้องมี:
```text
third_party/google/api/annotations.proto
third_party/google/api/http.proto
```

> ถ้า generate ไม่ได้เพราะหาไฟล์ไม่เจอ ให้เช็ก path `-I third_party` ใน Makefile

---

## 4) Define Proto with HTTP Annotations

สร้างไฟล์ `proto/user/v1/user.proto`

```proto
syntax = "proto3";

package user.v1;

option go_package = "github.com/ChinawatDc/go-api-grpc-gateway/gen/go/user/v1;userv1";

import "google/api/annotations.proto";

message User {
  string id = 1;
  string email = 2;
  string name = 3;
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

message CreateUserRequest {
  string email = 1;
  string name = 2;
}

message CreateUserResponse {
  User user = 1;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }

  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "*"
    };
  }
}
```

---

## 5) Generate Code (proto + grpc + gateway + openapi)

### 5.1 Makefile

สร้าง `Makefile`:

```makefile
.PHONY: proto tidy run-grpc run-gw

PROTO_DIR=proto
GEN_DIR=gen/go
THIRD_PARTY=third_party
OPENAPI_DIR=openapi

proto:
	@echo ">> generating protobuf + grpc + gateway + openapi..."
	protoc -I $(PROTO_DIR) -I $(THIRD_PARTY) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(GEN_DIR) --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=$(OPENAPI_DIR) \
		$(PROTO_DIR)/user/v1/user.proto
	@echo ">> done"

tidy:
	go mod tidy

run-grpc:
	go run ./cmd/grpc-server

run-gw:
	go run ./cmd/gateway
```

### 5.2 Run generate
```bash
make proto
make tidy
```

ผลลัพธ์ที่คาดหวัง:
```text
gen/go/user/v1/user.pb.go
gen/go/user/v1/user_grpc.pb.go
gen/go/user/v1/user.pb.gw.go
openapi/...
```

---

## 6) Business Logic (เหมือนบท go-api-grpc)

`internal/user/store.go`
```go
package user

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("user not found")

type Store interface {
	Get(id string) (User, error)
	Create(u User) (User, error)
}

type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]User
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]User)}
}

func (s *InMemoryStore) Get(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.items[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (s *InMemoryStore) Create(u User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[u.ID] = u
	return u, nil
}
```

`internal/user/service.go`
```go
package user

import "github.com/google/uuid"

type User struct {
	ID    string
	Email string
	Name  string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetUser(id string) (User, error) {
	return s.store.Get(id)
}

func (s *Service) CreateUser(email, name string) (User, error) {
	u := User{
		ID:    uuid.NewString(),
		Email: email,
		Name:  name,
	}
	return s.store.Create(u)
}
```

---

## 7) gRPC Server

`internal/transport/grpcserver/handler.go`
```go
package grpcserver

import (
	"context"

	userv1 "github.com/ChinawatDc/go-api-grpc-gateway/gen/go/user/v1"
	"github.com/ChinawatDc/go-api-grpc-gateway/internal/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	userv1.UnimplementedUserServiceServer
	svc *user.Service
}

func NewHandler(svc *user.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	u, err := h.svc.GetUser(req.GetId())
	if err != nil {
		if err == user.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &userv1.GetUserResponse{
		User: &userv1.User{Id: u.ID, Email: u.Email, Name: u.Name},
	}, nil
}

func (h *Handler) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	if req.GetEmail() == "" || req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and name are required")
	}
	u, err := h.svc.CreateUser(req.GetEmail(), req.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &userv1.CreateUserResponse{
		User: &userv1.User{Id: u.ID, Email: u.Email, Name: u.Name},
	}, nil
}
```

`internal/transport/grpcserver/server.go`
```go
package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	userv1 "github.com/ChinawatDc/go-api-grpc-gateway/gen/go/user/v1"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	addr       string
}

func New(addr string, handler userv1.UserServiceServer, opts ...grpc.ServerOption) *Server {
	s := grpc.NewServer(opts...)
	userv1.RegisterUserServiceServer(s, handler)
	return &Server{grpcServer: s, addr: addr}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(ch)
	}()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	}
}

func DefaultStopContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
```

`cmd/grpc-server/main.go`
```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChinawatDc/go-api-grpc-gateway/internal/transport/grpcserver"
	"github.com/ChinawatDc/go-api-grpc-gateway/internal/user"
)

func main() {
	addr := ":50051"

	store := user.NewInMemoryStore()
	svc := user.NewService(store)
	handler := grpcserver.NewHandler(svc)

	srv := grpcserver.New(addr, handler)

	go func() {
		log.Println("gRPC server listening on", addr)
		if err := srv.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := grpcserver.DefaultStopContext()
	defer cancel()
	_ = srv.Stop(ctx)
	log.Println("gRPC server stopped")
}
```

---

## 8) grpc-gateway HTTP Server

`internal/transport/gateway/gateway.go`
```go
package gateway

import (
	"context"
	"net/http"
	"time"

	userv1 "github.com/ChinawatDc/go-api-grpc-gateway/gen/go/user/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Config struct {
	HTTPAddr string
	GRPCAddr string
}

func NewHTTPServer(cfg Config) (*http.Server, error) {
	mux := runtime.NewServeMux()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := []grpc.DialOption{grpc.WithInsecure()}

	if err := userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, cfg.GRPCAddr, opts); err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return srv, nil
}
```

`cmd/gateway/main.go`
```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChinawatDc/go-api-grpc-gateway/internal/transport/gateway"
)

func main() {
	cfg := gateway.Config{
		HTTPAddr: ":8080",
		GRPCAddr: "localhost:50051",
	}

	srv, err := gateway.NewHTTPServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("HTTP gateway listening on", cfg.HTTPAddr, "-> gRPC", cfg.GRPCAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("HTTP gateway stopped")
}
```

---

## 9) Run & Test

Generate:
```bash
make proto
make tidy
```

Run gRPC:
```bash
make run-grpc
```

Run gateway:
```bash
make run-gw
```

Test:
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@example.com","name":"Dev One"}'
```

---

## 10) Production Best Practices

- Production ห้ามใช้ `WithInsecure()` ให้ใช้ TLS/mTLS
- forward auth header -> gRPC metadata
- ใส่ gRPC interceptors (logging/metrics/tracing/auth)
- proto เป็น source of truth อย่าให้ REST drift

---

MIT License
# 010-go-api-grpc-gateway
