package etcd

import (
	"context"

	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"

	"github.com/1055373165/groupcache/consistenthash"
	pb "github.com/1055373165/groupcache/groupcachepb"
	"etcd/logger"
	serverregistrydiscover "etcd/server_registry_discover"
	"etcd/utils"
)

// server 模块为 groupcache 之间提供了通信能力
// 这样部署在其他机器上的 groupcache 可以通过访问 server 获取缓存
// 至于找哪一个主机，由一致性 hash 负责
const (
	defaultAddr     = "127.0.0.1:6324"
	defaultReplicas = 50
)

var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// server 和 Group 是解耦合的，所以 server 要自己实现并发控制
type Server struct {
	pb.UnimplementedGroupCacheServer

	Addr        string     // format: ip:port
	Status      bool       // true: running false: stop
	stopsSignal chan error // 通知 registery revoke 服务
	mu          sync.Mutex
	consHash    *consistenthash.ConsistentHash
	clients     map[string]*client
}

// NewServer 创建 cache 的 server，若 addr 为空，则使用 defaultAddr
func NewServer(addr string) (*Server, error) {
	if addr == "" {
		addr = defaultAddr
	}

	if !utils.ValidPerrAddr(addr) {
		return nil, fmt.Errorf("invalid addr %s, it should be x.x.x.x:port", addr)
	}
	return &Server{Addr: addr}, nil
}

// Get 实现了 Groupcache service 的 Get 方法
func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	group, key := req.GetGroup(), req.GetKey()
	resp := &pb.GetResponse{}
	logger.Logger.Info("[groupcache server %s] Recv RPC Request - (%s)/(%s)", s.Addr, group, key)

	if key == "" || group == "" {
		return resp, fmt.Errorf("key and group name is reqiured")
	}

	g := GetGroup(group)
	if g == nil {
		return resp, fmt.Errorf("group %s not found", group)
	}
	view, err := g.Get(key)
	if err != nil {
		return resp, err
	}

	resp.Value = view.Bytes()
	return resp, nil
}

// Start 启动 Cache 服务
func (s *Server) Start() error {
	s.mu.Lock()

	if s.Status {
		s.mu.Unlock()
		return fmt.Errorf("server %s is already started", s.Addr)
	}

	// ------------启动服务----------------
	// 1. 设置 status = true 表示服务器已经在运行
	// 2. 初始化 stop channel，用于通知 registry stop keepalive
	// 3. 初始化 tcp socket 并开始监听
	// 4. 注册 rpc 服务至 grpc，这样 grpc 收到 request 可以分发给 server 处理
	// 5. 将自己的服务名/Host地址注册至 etcd，这样 client 就可以通过 etcd 获取服务 Host 地址进行通信；这样做的好处是：client 只需要知道服务名称以及 etcd 的 Host 就可以获取
	// 指定服务的 IP，无需将它们写死在 client 代码中
	s.Status = true
	s.stopsSignal = make(chan error)

	port := strings.Split(s.Addr, ":")[1]
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen %s, error: %v", s.Addr, err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGroupCacheServer(grpcServer, s)

	// 注册服务至 etcd
	go func() {
		// Register never return unless stop signal received (blocked)
		err := serverregistrydiscover.Register("groupcache", s.Addr, s.stopsSignal)
		if err != nil {
			logger.Logger.Error(err.Error())
		}
		// close channel
		close(s.stopsSignal)
		// close tcp listen
		err = lis.Close()
		if err != nil {
			logger.Logger.Error(err.Error())
		}
		logger.Logger.Infof("[%s] Revoke service and close tcp socket ok.", s.Addr)
	}()

	logger.Logger.Infof("[%s] register service ok\n", s.Addr)
	s.mu.Unlock()
	// Serve接受侦听器列表上的传入连接，为每个连接创建一个新的ServerTransport和服务Goroutine。
	// 服务Goroutines读取GRPC请求，然后调用注册的处理程序来回复它们。当lis.Accept失败并出现致命错误时，Serve返回。当此方法返回时，LIS将关闭。
	// 除非调用Stop或GracefulStop，否则SERVE将返回非零错误。
	if err := grpcServer.Serve(lis); s.Status && err != nil {
		return fmt.Errorf("failed to serve %s, error: %v", s.Addr, err)
	}
	return nil
}

// SetPeers 将各个远端主机 IP 配置到 Server 里
// 这样 Server 就可以 Pick 它们了
// 注意：此操作是覆写操作，peersIP 必须满足 x.x.x.x:port 的格式
func (s *Server) SetPeers(peersAddr []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consHash = consistenthash.NewConsistentHash(defaultReplicas, nil)
	s.consHash.AddTruthNode(peersAddr...)
	s.clients = make(map[string]*client)

	for _, peersAddr := range peersAddr {
		if !utils.ValidPerrAddr(peersAddr) {
			panic(fmt.Sprintf("[peer %s] invalid address format, it shoulb be x.x.x.x:port", peersAddr))
		}
		// groupcache/localhost:8000
		service := fmt.Sprintf("groupcache/%s", peersAddr)
		// client {name string}  (c *client) Fetch(key string) ([]byte, error)
		s.clients[peersAddr] = NewClient(service)
	}
}

// Pick 根据一致性哈希选举出 key 应该存放在的 cache
// return false 代表从本地获取 cache
func (s *Server) Pick(key string) (Fetcher, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	peerAddr := s.consHash.GetTruthNode(key)
	// Pick itself
	if peerAddr == s.Addr {
		logger.Logger.Infof("oohhh! pick myself, i am %s\n", s.Addr)
		return nil, false
	}

	logger.Logger.Info("[cache %s] pick remote peer: %s\n", s.Addr, peerAddr)
	return s.clients[peerAddr], true
}

// Stop 停止 server 运行，如果 server 没有运行，这将是一个 no-op
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Status {
		return
	}
	// 发送停止 keepAlive 的信号，因为该节点要退出了，不需要再发送心跳探测了
	s.stopsSignal <- nil
	s.Status = false
	s.clients = nil // 清空一致性哈希信息，帮助 GC 进行垃圾回收
	s.consHash = nil
}

// 测试 Server 是否实现了 Picker 接口
var _ Picker = (*Server)(nil)
